package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebRouterStaticFallbackContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	buildFS := fstest.MapFS{
		"web/default/dist/static/js/app.0123abcdef.js": {
			Data: []byte("console.log('ok')"),
		},
	}
	indexPage := []byte("<!doctype html><div id=\"root\"></div>")
	router := gin.New()
	SetWebRouter(router, ThemeAssets{
		DefaultBuildFS:   buildFS,
		DefaultIndexPage: indexPage,
	})

	t.Run("serves an existing hashed asset", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/static/js/app.0123abcdef.js", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "console.log('ok')", recorder.Body.String())
		assert.Equal(t, "public, max-age=31536000, immutable", recorder.Header().Get("Cache-Control"))
	})

	t.Run("returns a real 404 for a missing hashed asset", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/static/js/missing.0123abcdef.js", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		assert.NotContains(t, recorder.Header().Get("Content-Type"), "text/html")
		assert.NotEqual(t, string(indexPage), recorder.Body.String())
	})

	t.Run("keeps SPA fallback for application routes", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/dashboard/example", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, string(indexPage), recorder.Body.String())
		assert.Equal(t, "no-cache, must-revalidate", recorder.Header().Get("Cache-Control"))
	})
}

func TestWebRouterPublicSSRContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalSetup := constant.Setup
	originalSystemName := common.SystemName
	originalLogo := common.Logo
	originalServerAddress := system_setting.ServerAddress
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"HeaderNavModules": `{"docs":{"enabled":true}}`,
		"HomePageContent":  "",
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		constant.Setup = originalSetup
		common.SystemName = originalSystemName
		common.Logo = originalLogo
		system_setting.ServerAddress = originalServerAddress
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	constant.Setup = true
	common.SystemName = `Test </script><b> Gateway`
	common.Logo = `https://cdn.example/logo.png?x=1&y=2`
	system_setting.ServerAddress = "https://api.example.com/"

	page := func(locale, body string) []byte {
		return []byte(`<!doctype html><html lang="` + locale + `"><head><title>All Token API</title><meta name="title" content="All Token API" /><!--new-api-bootstrap--></head><body>` + body + ` __NEW_API_SYSTEM_NAME__ __NEW_API_LOGO__ __NEW_API_SERVER_ADDRESS__</body></html>`)
	}
	buildFS := fstest.MapFS{
		"web/default/dist/prerender/en/home.html": {
			Data: page("en", "english-home"),
		},
		"web/default/dist/prerender/zhCN/home.html": {
			Data: page("zhCN", "chinese-home"),
		},
		"web/default/dist/prerender/zhCN/docs/tools/codex.html": {
			Data: page("zhCN", "chinese-codex"),
		},
		"web/default/dist/static/docs/manifest.json": {
			Data: []byte(`{"version":1,"locales":{"zhCN":{"/docs":"introduction.0123abcdef.json","/docs/tools/codex":"codex.0123abcdef.json"}}}`),
		},
	}
	indexPage := []byte("<!doctype html><div id=\"root\"></div>")
	engine := gin.New()
	SetWebRouter(engine, ThemeAssets{
		DefaultBuildFS:   buildFS,
		DefaultIndexPage: indexPage,
	})

	t.Run("selects locale and safely injects public bootstrap", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "chinese-home")
		assert.Contains(t, body, `Test &lt;/script&gt;&lt;b&gt; Gateway`)
		assert.Contains(t, body, `"home_page_content_loaded":true`)
		assert.Contains(t, body, `"locale":"zhCN"`)
		assert.Contains(t, body, `Test \u003c/script\u003e\u003cb\u003e Gateway`)
		assert.NotContains(t, body, `window.__NEW_API_PUBLIC_BOOTSTRAP__={"home_page_content":"","home_page_content_loaded":true,"locale":"zhCN","setup":true,"status":{"HeaderNavModules":"{\"docs\":{\"enabled\":true}}","logo":"https://cdn.example/logo.png?x=1\u0026y=2","server_address":"https://api.example.com","system_name":"Test </script>`)
		assert.Equal(t, "no-cache, must-revalidate", recorder.Header().Get("Cache-Control"))
		assert.Equal(t, "Cookie, Accept-Language", recorder.Header().Get("Vary"))
	})

	t.Run("language cookie overrides request header", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: "i18next", Value: "en"})
		request.Header.Set("Accept-Language", "zh-CN")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "english-home")
	})

	t.Run("serves a prerendered docs route", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/docs/tools/codex", nil)
		request.AddCookie(&http.Cookie{Name: "i18next", Value: "en"})
		request.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, "chinese-codex")
		assert.Contains(t, body, `"home_page_content_loaded":false`)
		assert.Contains(t, body, `"file_name":"codex.0123abcdef.json"`)
		assert.Contains(t, body, `"locale":"zhCN"`)
	})

	t.Run("serves the docs manifest with conditional caching", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/static/docs/manifest.json", nil)
		first := httptest.NewRecorder()
		engine.ServeHTTP(first, request)

		require.Equal(t, http.StatusOK, first.Code)
		etag := first.Header().Get("ETag")
		require.NotEmpty(t, etag)
		assert.Equal(t, "no-cache, must-revalidate", first.Header().Get("Cache-Control"))

		request = httptest.NewRequest(http.MethodGet, "/static/docs/manifest.json", nil)
		request.Header.Set("If-None-Match", etag)
		second := httptest.NewRecorder()
		engine.ServeHTTP(second, request)

		assert.Equal(t, http.StatusNotModified, second.Code)
		assert.Equal(t, etag, second.Header().Get("ETag"))
	})

	t.Run("keeps custom homepage on the SPA path", func(t *testing.T) {
		common.OptionMapRWMutex.Lock()
		common.OptionMap["HomePageContent"] = "https://example.com/custom-home"
		common.OptionMapRWMutex.Unlock()
		t.Cleanup(func() {
			common.OptionMapRWMutex.Lock()
			common.OptionMap["HomePageContent"] = ""
			common.OptionMapRWMutex.Unlock()
		})

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, string(indexPage), recorder.Body.String())
	})

	t.Run("redirects public pages while setup is incomplete", func(t *testing.T) {
		constant.Setup = false
		t.Cleanup(func() { constant.Setup = true })

		request := httptest.NewRequest(http.MethodGet, "/docs", nil)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
		assert.Equal(t, "/setup", recorder.Header().Get("Location"))
	})
}
