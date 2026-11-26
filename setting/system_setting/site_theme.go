package system_setting

import (
	"errors"
	"fmt"
	"slices"

	"github.com/QuantumNous/new-api/common"
)

const SiteThemeCustomizationOptionKey = "SiteThemeCustomization"

type SiteThemeCustomization struct {
	Preset        string `json:"preset"`
	Font          string `json:"font"`
	Radius        string `json:"radius"`
	Scale         string `json:"scale"`
	ContentLayout string `json:"contentLayout"`
	Mode          string `json:"mode"`
	Sidebar       string `json:"sidebar"`
	Layout        string `json:"layout"`
	Direction     string `json:"direction"`
}

func DefaultSiteThemeCustomization() SiteThemeCustomization {
	return SiteThemeCustomization{
		Preset:        "default",
		Font:          "default",
		Radius:        "default",
		Scale:         "default",
		ContentLayout: "full",
		Mode:          "system",
		Sidebar:       "inset",
		Layout:        "default",
		Direction:     "ltr",
	}
}

func ParseSiteThemeCustomization(raw string) (SiteThemeCustomization, error) {
	var fields map[string]any
	if err := common.UnmarshalJsonStr(raw, &fields); err != nil || fields == nil {
		return SiteThemeCustomization{}, errors.New("站点主题必须为 JSON 对象")
	}

	theme := DefaultSiteThemeCustomization()
	for name, value := range fields {
		var field *string
		var allowed []string
		switch name {
		case "preset":
			field, allowed = &theme.Preset, []string{
				"default", "anthropic", "simple-large", "underground", "rose-garden", "lake-view",
				"sunset-glow", "forest-whisper", "ocean-breeze", "lavender-dream", "ikun-love",
			}
		case "font":
			field, allowed = &theme.Font, []string{"default", "sans", "serif"}
		case "radius":
			field, allowed = &theme.Radius, []string{"default", "none", "sm", "md", "lg", "xl"}
		case "scale":
			field, allowed = &theme.Scale, []string{"default", "sm", "lg", "xl"}
		case "contentLayout":
			field, allowed = &theme.ContentLayout, []string{"full", "centered"}
		case "mode":
			field, allowed = &theme.Mode, []string{"system", "light", "dark"}
		case "sidebar":
			field, allowed = &theme.Sidebar, []string{"inset", "floating", "sidebar"}
		case "layout":
			field, allowed = &theme.Layout, []string{"default", "icon", "offcanvas"}
		case "direction":
			field, allowed = &theme.Direction, []string{"ltr", "rtl"}
		default:
			return SiteThemeCustomization{}, fmt.Errorf("未知站点主题字段 %q", name)
		}
		option, ok := value.(string)
		if !ok || !slices.Contains(allowed, option) {
			return SiteThemeCustomization{}, fmt.Errorf("站点主题字段 %q 的值无效", name)
		}
		*field = option
	}
	return theme, nil
}

func (theme SiteThemeCustomization) ResolvedFont() string {
	if theme.Font != "default" {
		return theme.Font
	}
	if theme.Preset == "anthropic" {
		return "serif"
	}
	return "sans"
}
