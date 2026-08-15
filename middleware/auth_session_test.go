package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func performSessionAuthRequest(t *testing.T, auth gin.HandlerFunc, role int, includeSession bool, includeUserHeader bool) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("auth-session-test"))))
	router.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "tester")
		session.Set("role", role)
		session.Set("id", 42)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET("/protected", auth, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	var cookies []*http.Cookie
	if includeSession {
		loginRecorder := httptest.NewRecorder()
		router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
		require.Equal(t, http.StatusNoContent, loginRecorder.Code)
		cookies = loginRecorder.Result().Cookies()
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	for _, sessionCookie := range cookies {
		request.AddCookie(sessionCookie)
	}
	if includeUserHeader {
		request.Header.Set("New-Api-User", "42")
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminSessionAuthAllowsAdminSessionWithoutUserHeader(t *testing.T) {
	recorder := performSessionAuthRequest(t, AdminSessionAuth(), common.RoleAdminUser, true, false)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAdminSessionAuthRejectsMissingSession(t *testing.T) {
	recorder := performSessionAuthRequest(t, AdminSessionAuth(), common.RoleAdminUser, false, false)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAdminSessionAuthRejectsCommonUserSession(t *testing.T) {
	recorder := performSessionAuthRequest(t, AdminSessionAuth(), common.RoleCommonUser, true, false)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
}

func TestAdminAuthStillRequiresUserHeader(t *testing.T) {
	recorder := performSessionAuthRequest(t, AdminAuth(), common.RoleAdminUser, true, false)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAdminAuthAllowsMatchingUserHeader(t *testing.T) {
	recorder := performSessionAuthRequest(t, AdminAuth(), common.RoleAdminUser, true, true)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}
