package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchUpstreamRatiosRejectsUnavailableOrInvalidPrices(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		error  string
	}{
		{"missing preset", http.StatusNotFound, `not found`, "404"},
		{"unsupported variable", http.StatusOK, `{"success":true,"data":{"billing_mode":{"gpt-test":"tiered_expr"},"billing_expr":{"gpt-test":"tier(\"base\", unsupported_cache * 2)"}}}`, "unsupported_cache"},
		{"missing expression", http.StatusOK, `{"success":true,"data":{"billing_mode":{"gpt-test":"tiered_expr"}}}`, "gpt-test"},
		{"non-finite price", http.StatusOK, `{"success":true,"data":{"billing_expr":{"gpt-test":"tier(\"base\", 1e308 * 1e308)"}}}`, "invalid cost"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			requestBody, err := common.Marshal(map[string]any{
				"upstreams": []map[string]any{{"id": -100, "name": "官方倍率预设", "base_url": server.URL, "endpoint": "/"}},
			})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ratio_sync/fetch", strings.NewReader(string(requestBody)))
			FetchUpstreamRatios(ctx)
			var payload struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.False(t, payload.Success)
			assert.Contains(t, payload.Message, test.error)
		})
	}
}

func TestUpstreamBillingAcceptsCurrentPresetContracts(t *testing.T) {
	data := map[string]any{
		"billing_mode": map[string]string{"gpt-5.6-terra": "tiered_expr", "image-01": "tiered_expr", "deepseek-flash": "tiered_expr"},
		"billing_expr": map[string]string{
			"gpt-5.6-terra":  `len <= 272000 ? tier("0_272k", p * 2 + cr * 0.2 + cc * 2.5 + cc12h * 5 + c * 12) : tier("272k_plus", p * 4 + cr * 0.4 + cc * 5 + cc12h * 10 + c * 18)`,
			"image-01":       `tier("image", fixed(0.0035)) * image_count`,
			"deepseek-flash": `weekday("UTC") >= 1 && weekday("UTC") <= 5 && ((hour("UTC") >= 1 && hour("UTC") < 4) || (hour("UTC") >= 6 && hour("UTC") < 10)) ? tier("peak", p * 0.3 + cr * 0.006 + c * 1.2) : tier("off_peak", p * 0.15 + cr * 0.003 + c * 0.6)`,
		},
	}
	require.NoError(t, validateUpstreamBilling(data))
	channels := []struct {
		name string
		data map[string]any
	}{{name: "官方倍率预设", data: data}}
	assert.Empty(t, buildDifferences(data, channels))
}

func TestApplyUpstreamPricesRejectsInvalidBatchBeforeWriting(t *testing.T) {
	for _, updates := range []map[string]string{
		{"PaymentToken": `"secret"`},
		{"ModelRatio": `{"gpt-test":-1}`},
		{"billing_setting.billing_mode": `{"gpt-test":"tiered_expr"}`, "billing_setting.billing_expr": `{}`},
		{"billing_setting.billing_expr": `{"gpt-test":"tier(\"base\", wrong_variable)"}`, "ModelRatio": `{"gpt-test":1}`},
	} {
		body, err := common.Marshal(map[string]any{"updates": updates})
		require.NoError(t, err)
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ratio_sync/apply", strings.NewReader(string(body)))
		before := billing_setting.GetBillingExprCopy()
		ApplyUpstreamPrices(ctx)
		var payload struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		assert.False(t, payload.Success)
		assert.Equal(t, before, billing_setting.GetBillingExprCopy())
	}
}
