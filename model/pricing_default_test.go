package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInferDefaultVendor(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      string
	}{
		{
			name:      "gpt codex spark prefers OpenAI",
			modelName: "gpt-5.3-codex-spark",
			want:      "OpenAI",
		},
		{
			name:      "spark model uses iFlyTek",
			modelName: "spark-4.0",
			want:      "讯飞",
		},
		{
			name:      "matching is case insensitive",
			modelName: "GPT-4.1",
			want:      "OpenAI",
		},
		{
			name:      "unknown model has no default vendor",
			modelName: "custom-model",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, inferDefaultVendor(tt.modelName))
		})
	}
}
