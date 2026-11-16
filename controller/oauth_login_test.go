package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthLoginStartsOnCallbackOrigin(t *testing.T) {
	require.NoError(t, i18n.Init())
	previousAddress, previousEnabled, previousClientID := system_setting.ServerAddress, common.GitHubOAuthEnabled, common.GitHubClientId
	t.Cleanup(func() {
		system_setting.ServerAddress, common.GitHubOAuthEnabled, common.GitHubClientId = previousAddress, previousEnabled, previousClientID
	})
	system_setting.ServerAddress = "https://example.com"
	common.GitHubOAuthEnabled, common.GitHubClientId = true, "test-client"
	router := gin.New()
	store := cookie.NewStore([]byte("oauth-login-test-key-32-bytes-long"))
	store.Options(sessions.Options{Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	router.Use(sessions.Sessions("session", store))
	router.GET("/api/oauth/start/:provider", StartOAuthLogin)
	router.GET("/api/oauth/:provider", HandleOAuth)
	router.GET("/session", func(c *gin.Context) {
		session := sessions.Default(c)
		common.ApiSuccess(c, session.Get("aff"))
	})
	router.GET("/expired-session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "expired-state")
		session.Set("oauth_started_at", common.GetTimestamp()-601)
		require.NoError(t, session.Save())
	})

	// An explicit invitation code travels to the main origin before a session
	// is issued. No cross-subdomain session cookie is needed.
	subdomain := httptest.NewRecorder()
	router.ServeHTTP(subdomain, httptest.NewRequest(http.MethodGet, "https://sales.example.com/api/oauth/start/github?aff=EXPLICIT", nil))
	require.Equal(t, http.StatusSeeOther, subdomain.Code)
	assert.Equal(t, "https://example.com/api/oauth/start/github?aff=EXPLICIT", subdomain.Header().Get("Location"))
	assert.Empty(t, subdomain.Result().Cookies())

	main := httptest.NewRecorder()
	router.ServeHTTP(main, httptest.NewRequest(http.MethodGet, subdomain.Header().Get("Location"), nil))
	require.Equal(t, http.StatusSeeOther, main.Code)
	authorization, err := url.Parse(main.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "github.com", authorization.Host)
	assert.Equal(t, "https://example.com/oauth/github", authorization.Query().Get("redirect_uri"))
	state := authorization.Query().Get("state")
	require.NotEmpty(t, state)
	cookies := main.Result().Cookies()
	require.NotEmpty(t, cookies)
	assert.Empty(t, cookies[0].Domain)
	assert.True(t, cookies[0].HttpOnly)

	request := httptest.NewRequest(http.MethodGet, "https://example.com/session", nil)
	request.AddCookie(cookies[0])
	sessionResponse := httptest.NewRecorder()
	router.ServeHTTP(sessionResponse, request)
	assert.Contains(t, sessionResponse.Body.String(), `"data":"EXPLICIT"`)

	// A denied provider callback still consumes the state without contacting
	// GitHub, and the resulting session rejects a second callback.
	callbackURL := "https://example.com/api/oauth/github?state=" + state + "&error=access_denied&error_description=access_denied"
	request = httptest.NewRequest(http.MethodGet, callbackURL, nil)
	request.AddCookie(cookies[0])
	callback := httptest.NewRecorder()
	router.ServeHTTP(callback, request)
	require.Equal(t, http.StatusOK, callback.Code)
	assert.Contains(t, callback.Body.String(), "access_denied")
	updatedCookies := callback.Result().Cookies()
	require.NotEmpty(t, updatedCookies)
	request = httptest.NewRequest(http.MethodGet, callbackURL, nil)
	request.AddCookie(updatedCookies[0])
	repeated := httptest.NewRecorder()
	router.ServeHTTP(repeated, request)
	assert.Equal(t, http.StatusForbidden, repeated.Code)

	expiredSession := httptest.NewRecorder()
	router.ServeHTTP(expiredSession, httptest.NewRequest(http.MethodGet, "https://example.com/expired-session", nil))
	expiredCookies := expiredSession.Result().Cookies()
	require.NotEmpty(t, expiredCookies)
	request = httptest.NewRequest(http.MethodGet, "https://example.com/api/oauth/github?state=expired-state&error=access_denied&error_description=access_denied", nil)
	request.AddCookie(expiredCookies[0])
	expired := httptest.NewRecorder()
	router.ServeHTTP(expired, request)
	assert.Contains(t, expired.Body.String(), `"success":false`)
	assert.NotContains(t, expired.Body.String(), "access_denied")
}
