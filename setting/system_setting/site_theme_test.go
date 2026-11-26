package system_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSiteThemeCustomization(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  string
		want SiteThemeCustomization
	}{
		{
			name: "defaults",
			raw:  `{}`,
			want: DefaultSiteThemeCustomization(),
		},
		{
			name: "existing configuration uses defaults for new fields",
			raw:  `{"preset":"ikun-love","font":"sans","radius":"none","scale":"lg","contentLayout":"centered"}`,
			want: SiteThemeCustomization{
				Preset: "ikun-love", Font: "sans", Radius: "none", Scale: "lg", ContentLayout: "centered",
				Mode: "system", Sidebar: "inset", Layout: "default", Direction: "ltr",
			},
		},
		{
			name: "configured values",
			raw:  `{"preset":"anthropic","font":"serif","radius":"none","scale":"xl","contentLayout":"centered","mode":"dark","sidebar":"floating","layout":"offcanvas","direction":"rtl"}`,
			want: SiteThemeCustomization{
				Preset: "anthropic", Font: "serif", Radius: "none", Scale: "xl", ContentLayout: "centered",
				Mode: "dark", Sidebar: "floating", Layout: "offcanvas", Direction: "rtl",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseSiteThemeCustomization(test.raw)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

	for _, raw := range []string{
		``, `[]`, `null`, `true`, `{} {}`, `{"unknown":"default"}`, `{"font":null}`, `{"radius":12}`,
		`{"preset":"other"}`, `{"font":"mono"}`, `{"radius":"xs"}`, `{"scale":"md"}`, `{"contentLayout":"wide"}`,
		`{"mode":"auto"}`, `{"sidebar":"fixed"}`, `{"layout":"none"}`, `{"direction":"right"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := ParseSiteThemeCustomization(raw)
			require.Error(t, err)
		})
	}
}
