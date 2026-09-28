package groupbench

import (
	"bytes"
	"strings"

	"github.com/tidwall/gjson"
)

type ResponseText struct {
	Text       string
	StopReason string
	Model      string
}

// ExtractResponseText collects the answer text from a client-format response
// body. It accepts OpenAI chat completions, Anthropic messages, and OpenAI
// responses, either as one JSON document or as an SSE stream, and tells them
// apart by their fields rather than by the requested endpoint.
func ExtractResponseText(body []byte) ResponseText {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' && gjson.ValidBytes(trimmed) {
		return extractDocumentText(gjson.ParseBytes(trimmed))
	}

	var out ResponseText
	var text strings.Builder
	for _, line := range bytes.Split(trimmed, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) || !gjson.ValidBytes(payload) {
			continue
		}
		event := gjson.ParseBytes(payload)
		switch event.Get("type").String() {
		case "content_block_delta":
			if event.Get("delta.type").String() == "text_delta" {
				text.WriteString(event.Get("delta.text").String())
			}
		case "message_start":
			setIfEmpty(&out.Model, event.Get("message.model").String())
		case "message_delta":
			setIfNotEmpty(&out.StopReason, event.Get("delta.stop_reason").String())
		case "response.output_text.delta":
			text.WriteString(event.Get("delta").String())
		case "response.completed", "response.incomplete":
			setIfEmpty(&out.Model, event.Get("response.model").String())
			out.StopReason = responsesStopReason(event.Get("response"))
		default:
			choice := event.Get("choices.0")
			if !choice.Exists() {
				continue
			}
			setIfEmpty(&out.Model, event.Get("model").String())
			text.WriteString(choice.Get("delta.content").String())
			setIfNotEmpty(&out.StopReason, choice.Get("finish_reason").String())
		}
	}
	out.Text = text.String()
	return out
}

func extractDocumentText(doc gjson.Result) ResponseText {
	out := ResponseText{Model: doc.Get("model").String()}
	var text strings.Builder
	switch {
	case doc.Get("choices").Exists():
		out.Text = doc.Get("choices.0.message.content").String()
		out.StopReason = doc.Get("choices.0.finish_reason").String()
		return out
	case doc.Get("stop_reason").Exists():
		for _, block := range doc.Get("content").Array() {
			if block.Get("type").String() == "text" {
				text.WriteString(block.Get("text").String())
			}
		}
		out.StopReason = doc.Get("stop_reason").String()
	case doc.Get("output").Exists():
		for _, item := range doc.Get("output").Array() {
			for _, part := range item.Get("content").Array() {
				if part.Get("type").String() == "output_text" {
					text.WriteString(part.Get("text").String())
				}
			}
		}
		out.StopReason = responsesStopReason(doc)
	}
	out.Text = text.String()
	return out
}

// responsesStopReason reports incomplete_details.reason (e.g.
// "max_output_tokens") for an incomplete response and the status otherwise.
func responsesStopReason(response gjson.Result) string {
	if reason := response.Get("incomplete_details.reason").String(); reason != "" {
		return reason
	}
	return response.Get("status").String()
}

func setIfEmpty(dst *string, value string) {
	if *dst == "" {
		*dst = value
	}
}

func setIfNotEmpty(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}
