package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTimeoutOption(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "relay timeout allows unlimited", key: "RelayTimeout", value: "0"},
		{name: "relay timeout minimum", key: "RelayTimeout", value: "-1", wantErr: true},
		{name: "relay timeout maximum", key: "RelayTimeout", value: "3601", wantErr: true},
		{name: "streaming timeout minimum", key: "StreamingTimeout", value: "0", wantErr: true},
		{name: "streaming timeout maximum", key: "StreamingTimeout", value: "3600"},
		{name: "idle connection allows unlimited", key: "RelayIdleConnTimeout", value: "0"},
		{name: "idle connection rejects negative", key: "RelayIdleConnTimeout", value: "-1", wantErr: true},
		{name: "client write timeout maximum", key: "StreamClientWriteTimeout", value: "600"},
		{name: "client write timeout over maximum", key: "StreamClientWriteTimeout", value: "601", wantErr: true},
		{name: "shutdown grace minimum", key: "ShutdownTimeoutSeconds", value: "1"},
		{name: "shutdown grace over maximum", key: "ShutdownTimeoutSeconds", value: "901", wantErr: true},
		{name: "non integer", key: "StreamingTimeout", value: "30.5", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeoutOption(tt.key, tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
