package constant

import "testing"

func TestPath2RelayModeAlphaSearch(t *testing.T) {
	if got := Path2RelayMode("/v1/alpha/search"); got != RelayModeAlphaSearch {
		t.Fatalf("Path2RelayMode(/v1/alpha/search) = %d, want %d", got, RelayModeAlphaSearch)
	}
}
