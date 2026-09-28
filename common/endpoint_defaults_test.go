package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultEndpointInfoAlphaSearch(t *testing.T) {
	info, ok := GetDefaultEndpointInfo(constant.EndpointTypeOpenAIAlphaSearch)
	require.True(t, ok)
	require.Equal(t, "/v1/alpha/search", info.Path)
	require.Equal(t, "POST", info.Method)
}
