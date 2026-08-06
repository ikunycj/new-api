package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlaygroundRoutesShareGroupOverrideHandling(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "chat completions", path: "/pg/chat/completions", want: true},
		{name: "image generations", path: "/pg/images/generations", want: true},
		{name: "public image generations", path: "/v1/images/generations", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.path, nil)
			assert.Equal(t, test.want, isPlaygroundRequest(request.URL.Path))
		})
	}
}
