// Package groupbench defines the prompts a group bench round sends to every
// channel of a pricing group, and how their answers are turned into stored
// artifacts and metrics.
package groupbench

import (
	"regexp"
	"sort"
)

// MaxArtifactBytes keeps artifacts under MySQL's 64KB TEXT limit.
const MaxArtifactBytes = 60 << 10

// fallbackArtifactBytes is how much of the raw answer is kept when a preset
// finds nothing to extract, so the admin can still see what came back.
const fallbackArtifactBytes = 2 << 10

const DefaultPresetKey = "pelican_anim"

type Artifact struct {
	ContentType string
	Content     string
	Clipped     bool
}

// Preset is one bench question. Adding a new question only needs a new entry in
// presets; the storage and UI are keyed by ContentType, not by preset.
type Preset struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Prompt    string `json:"prompt"`
	MaxTokens uint   `json:"max_tokens"`
	// Extract pulls the artifact out of the answer text; ok=false means the
	// answer did not contain one.
	Extract func(text string) (artifact Artifact, ok bool) `json:"-"`
	// Analyze computes preset-specific metrics of an extracted artifact.
	Analyze func(content string) any `json:"-"`
}

var presets = map[string]*Preset{
	DefaultPresetKey: {
		Key:  DefaultPresetKey,
		Name: "Pelican riding a bicycle (SMIL animated SVG)",
		Prompt: "Generate an animated SVG of a pelican riding a bicycle. " +
			"Use SMIL <animateTransform> so the wheels spin, the pedals rotate, " +
			"and the pelican's wings flap. " +
			"Must animate on its own in a browser with no JS.",
		// Reasoning models spend part of the budget thinking; 8000 was enough
		// for the SVG alone in the relay_audit runs.
		MaxTokens: 16000,
		Extract:   ExtractSVG,
		Analyze:   func(content string) any { return AnalyzeSVG(content) },
	},
}

func GetPreset(key string) (*Preset, bool) {
	preset, ok := presets[key]
	return preset, ok
}

func ListPresets() []*Preset {
	list := make([]*Preset, 0, len(presets))
	for _, preset := range presets {
		list = append(list, preset)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Key < list[j].Key })
	return list
}

// FallbackArtifact keeps the head of an answer that had nothing to extract.
func FallbackArtifact(text string) Artifact {
	content, clipped := ClipUTF8(text, fallbackArtifactBytes)
	return Artifact{ContentType: "text/plain", Content: content, Clipped: clipped}
}

var (
	svgBlockRe     = regexp.MustCompile(`(?is)<svg\b.*?</svg>`)
	svgOpenRe      = regexp.MustCompile(`(?i)<svg\b`)
	svgElementRe   = regexp.MustCompile(`<[a-zA-Z]`)
	svgGroupRe     = regexp.MustCompile(`(?i)<g\b`)
	svgPathRe      = regexp.MustCompile(`(?i)<path\b`)
	svgAnimateRe   = regexp.MustCompile(`(?i)<animate\b`)
	svgAnimTransRe = regexp.MustCompile(`(?i)<animateTransform\b`)
	svgAnimMotion  = regexp.MustCompile(`(?i)<animateMotion\b`)
	svgRotateRe    = regexp.MustCompile(`(?i)type\s*=\s*["']rotate`)
	svgIndefRe     = regexp.MustCompile(`(?i)indefinite`)
	svgKeyframesRe = regexp.MustCompile(`(?i)@keyframes`)
	svgScriptRe    = regexp.MustCompile(`(?i)<script\b`)
	svgViewBoxRe   = regexp.MustCompile(`(?i)viewBox\s*=\s*["']([^"']+)`)
)

// ExtractSVG returns the first complete <svg>…</svg> block. An answer cut off
// before </svg> (e.g. max_tokens hit) yields the unterminated tail, flagged as
// clipped, so the partial drawing is still visible.
func ExtractSVG(text string) (Artifact, bool) {
	block := svgBlockRe.FindString(text)
	clipped := false
	if block == "" {
		loc := svgOpenRe.FindStringIndex(text)
		if loc == nil {
			return Artifact{}, false
		}
		block = text[loc[0]:]
		clipped = true
	}
	content, cut := ClipUTF8(block, MaxArtifactBytes)
	return Artifact{ContentType: "image/svg+xml", Content: content, Clipped: clipped || cut}, true
}

type SVGMetrics struct {
	Bytes            int    `json:"bytes"`
	Elements         int    `json:"elements"`
	Groups           int    `json:"groups"`
	Paths            int    `json:"paths"`
	Animate          int    `json:"animate"`
	AnimateTransform int    `json:"animate_transform"`
	AnimateMotion    int    `json:"animate_motion"`
	Rotate           int    `json:"rotate"`
	Indefinite       int    `json:"indefinite"`
	CSSKeyframes     int    `json:"css_keyframes"`
	HasScript        bool   `json:"has_script"`
	ViewBox          string `json:"view_box,omitempty"`
	// Animated is true when any SMIL or CSS animation is present.
	Animated bool `json:"animated"`
}

// AnalyzeSVG mirrors analyze_svg in relay_audit/channel_quality_ab.py so the
// numbers stay comparable with the offline A/B runs.
func AnalyzeSVG(svg string) SVGMetrics {
	metrics := SVGMetrics{
		Bytes:            len(svg),
		Elements:         len(svgElementRe.FindAllStringIndex(svg, -1)),
		Groups:           len(svgGroupRe.FindAllStringIndex(svg, -1)),
		Paths:            len(svgPathRe.FindAllStringIndex(svg, -1)),
		Animate:          len(svgAnimateRe.FindAllStringIndex(svg, -1)),
		AnimateTransform: len(svgAnimTransRe.FindAllStringIndex(svg, -1)),
		AnimateMotion:    len(svgAnimMotion.FindAllStringIndex(svg, -1)),
		Rotate:           len(svgRotateRe.FindAllStringIndex(svg, -1)),
		Indefinite:       len(svgIndefRe.FindAllStringIndex(svg, -1)),
		CSSKeyframes:     len(svgKeyframesRe.FindAllStringIndex(svg, -1)),
		HasScript:        svgScriptRe.MatchString(svg),
	}
	if m := svgViewBoxRe.FindStringSubmatch(svg); m != nil {
		metrics.ViewBox = m[1]
	}
	// `<animate\b` does not match <animateTransform>/<animateMotion>, because
	// \b needs a non-word char after "animate".
	metrics.Animated = metrics.Animate+metrics.AnimateTransform+metrics.AnimateMotion+metrics.CSSKeyframes > 0
	return metrics
}

// ClipUTF8 cuts s to at most limit bytes without splitting a UTF-8 sequence.
func ClipUTF8(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut], true
}
