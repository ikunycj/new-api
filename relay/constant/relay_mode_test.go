package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath2RelayModeRecognizesImageGenerationRoutes(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "public API", path: "/v1/images/generations"},
		{name: "playground", path: "/pg/images/generations"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, RelayModeImagesGenerations, Path2RelayMode(test.path))
		})
	}
}
