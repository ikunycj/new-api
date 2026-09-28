package groupbench

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const animatedSVG = `<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg">
<g id="wheel"><circle cx="50" cy="150" r="30"/>
<animateTransform attributeName="transform" type="rotate" from="0 50 150" to="360 50 150" dur="1s" repeatCount="indefinite"/></g>
<path d="M0 0"/><animate attributeName="opacity" values="0;1" dur="2s"/>
</svg>`

func TestExtractSVG(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantOK      bool
		wantClipped bool
		wantPrefix  string
		wantSuffix  string
	}{
		{name: "fenced block", text: "Here:\n```svg\n" + animatedSVG + "\n```\nDone.", wantOK: true, wantPrefix: "<svg", wantSuffix: "</svg>"},
		{name: "first of two", text: "<svg a></svg> and <svg b></svg>", wantOK: true, wantPrefix: "<svg a>", wantSuffix: "</svg>"},
		{name: "cut before close", text: "ok <svg viewBox=\"0 0 1 1\"><g>", wantOK: true, wantClipped: true, wantPrefix: "<svg", wantSuffix: "<g>"},
		{name: "no svg", text: "I cannot draw that.", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact, ok := ExtractSVG(tt.text)
			require.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.Equal(t, "image/svg+xml", artifact.ContentType)
			assert.Equal(t, tt.wantClipped, artifact.Clipped)
			assert.True(t, strings.HasPrefix(artifact.Content, tt.wantPrefix), artifact.Content)
			assert.True(t, strings.HasSuffix(artifact.Content, tt.wantSuffix), artifact.Content)
		})
	}
}

func TestExtractSVGClipsOversizedArtifact(t *testing.T) {
	huge := "<svg>" + strings.Repeat("鹈", MaxArtifactBytes) + "</svg>"
	artifact, ok := ExtractSVG(huge)
	require.True(t, ok)
	assert.True(t, artifact.Clipped)
	assert.LessOrEqual(t, len(artifact.Content), MaxArtifactBytes)
	assert.True(t, strings.HasSuffix(artifact.Content, "鹈"), "must not split a UTF-8 sequence")
}

func TestAnalyzeSVG(t *testing.T) {
	tests := []struct {
		name string
		svg  string
		want SVGMetrics
	}{
		{
			name: "smil animated",
			svg:  animatedSVG,
			want: SVGMetrics{Elements: 6, Groups: 1, Paths: 1, Animate: 1, AnimateTransform: 1,
				Rotate: 1, Indefinite: 1, ViewBox: "0 0 200 200", Animated: true},
		},
		{
			name: "static",
			svg:  `<svg><circle r="1"/></svg>`,
			want: SVGMetrics{Elements: 2},
		},
		{
			name: "css animation with script",
			svg:  `<svg><style>@keyframes flap{}</style><script>x()</script></svg>`,
			want: SVGMetrics{Elements: 3, CSSKeyframes: 1, HasScript: true, Animated: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Bytes = len(tt.svg)
			assert.Equal(t, tt.want, AnalyzeSVG(tt.svg))
		})
	}
}

func TestExtractResponseText(t *testing.T) {
	tests := []struct {
		name string
		body string
		want ResponseText
	}{
		{
			name: "chat completion",
			body: `{"model":"gpt-6-sol","choices":[{"message":{"content":"<svg/>"},"finish_reason":"stop"}]}`,
			want: ResponseText{Text: "<svg/>", StopReason: "stop", Model: "gpt-6-sol"},
		},
		{
			name: "chat stream",
			body: "data: {\"model\":\"gpt-6-sol\",\"choices\":[{\"delta\":{\"content\":\"<svg>\"}}]}\n\n" +
				"data: {\"model\":\"gpt-6-sol\",\"choices\":[{\"delta\":{\"content\":\"</svg>\"},\"finish_reason\":\"length\"}]}\n\n" +
				"data: {\"choices\":[],\"usage\":{\"completion_tokens\":3}}\n\ndata: [DONE]\n",
			want: ResponseText{Text: "<svg></svg>", StopReason: "length", Model: "gpt-6-sol"},
		},
		{
			name: "anthropic message",
			body: `{"model":"claude-opus-5","content":[{"type":"thinking","thinking":"hm"},{"type":"text","text":"<svg/>"}],"stop_reason":"end_turn"}`,
			want: ResponseText{Text: "<svg/>", StopReason: "end_turn", Model: "claude-opus-5"},
		},
		{
			name: "anthropic stream",
			body: "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-5\"}}\n\n" +
				"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"hm\"}}\n\n" +
				"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"<svg>\"}}\n\n" +
				"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"</svg>\"}}\n\n" +
				"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"max_tokens\"}}\n\n",
			want: ResponseText{Text: "<svg></svg>", StopReason: "max_tokens", Model: "claude-opus-5"},
		},
		{
			name: "responses document",
			body: `{"model":"gpt-6-sol","status":"completed","output":[{"type":"reasoning"},{"type":"message","content":[{"type":"output_text","text":"<svg/>"}]}]}`,
			want: ResponseText{Text: "<svg/>", StopReason: "completed", Model: "gpt-6-sol"},
		},
		{
			name: "responses stream incomplete",
			body: "data: {\"type\":\"response.output_text.delta\",\"delta\":\"<svg>\"}\n\n" +
				"data: {\"type\":\"response.incomplete\",\"response\":{\"model\":\"gpt-6-sol\",\"status\":\"incomplete\",\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n",
			want: ResponseText{Text: "<svg>", StopReason: "max_output_tokens", Model: "gpt-6-sol"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractResponseText([]byte(tt.body)))
		})
	}
}
