package router

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func prepareLogoImageRouterStorage(t *testing.T) []byte {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL logo image route tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.Close()) })
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { assert.NoError(t, tx.Rollback().Error) })
	schema := fmt.Sprintf("logo_image_route_test_%d", time.Now().UnixNano())
	require.NoError(t, tx.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	require.NoError(t, tx.Exec(`SET LOCAL search_path TO "`+schema+`"`).Error)
	require.NoError(t, tx.AutoMigrate(&model.Option{}))
	previousDB, previousType := model.DB, common.MainDatabaseType()
	model.DB = tx
	common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	t.Cleanup(func() {
		assert.NoError(t, model.DeleteLogoImage("existing.png"))
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	previousRedis := common.RedisEnabled
	previousAPIRateLimit := common.GlobalApiRateLimitEnable
	previousWebRateLimit := common.GlobalWebRateLimitEnable
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = false
	common.GlobalWebRateLimitEnable = false
	t.Cleanup(func() {
		common.RedisEnabled = previousRedis
		common.GlobalApiRateLimitEnable = previousAPIRateLimit
		common.GlobalWebRateLimitEnable = previousWebRateLimit
	})

	var content bytes.Buffer
	require.NoError(t, png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	require.NoError(t, model.AddLogoImage("existing.png", content.Bytes()))
	return content.Bytes()
}

func logoRouterSessionCookies(t *testing.T, engine *gin.Engine, role int) []*http.Cookie {
	t.Helper()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test/sign-in/"+strconv.Itoa(role), nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	cookies := response.Result().Cookies()
	require.NotEmpty(t, cookies)
	return cookies
}

func TestLogoImageManagementRouterRequiresRoot(t *testing.T) {
	content := prepareLogoImageRouterStorage(t)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("logo-image-router-test"))))
	engine.GET("/test/sign-in/:role", func(c *gin.Context) {
		role, err := strconv.Atoi(c.Param("role"))
		require.NoError(t, err)
		session := sessions.Default(c)
		session.Set("username", "logo-test-user")
		session.Set("role", role)
		session.Set("id", 42)
		session.Set("status", common.UserStatusEnabled)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)

	request := httptest.NewRequest(http.MethodGet, "/api/option/logo-images", nil)
	request.Header.Set("New-Api-User", "42")
	for _, sessionCookie := range logoRouterSessionCookies(t, engine, common.RoleRootUser) {
		request.AddCookie(sessionCookie)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	var listing struct {
		Success bool `json:"success"`
		Data    []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &listing))
	require.True(t, listing.Success)
	require.Len(t, listing.Data, 1)
	assert.Equal(t, "existing.png", listing.Data[0].Name)
	assert.Equal(t, "/logo/existing.png", listing.Data[0].URL)

	for _, identity := range []struct {
		name   string
		role   int
		status int
	}{
		{"anonymous", 0, http.StatusUnauthorized},
		{"user", common.RoleCommonUser, http.StatusOK},
		{"administrator", common.RoleAdminUser, http.StatusOK},
	} {
		t.Run(identity.name, func(t *testing.T) {
			var cookies []*http.Cookie
			if identity.role != 0 {
				cookies = logoRouterSessionCookies(t, engine, identity.role)
			}
			for _, operation := range []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/api/option/logo-images"},
				{http.MethodPost, "/api/option/logo-images"},
				{http.MethodDelete, "/api/option/logo-images/existing.png"},
			} {
				var body bytes.Buffer
				request := httptest.NewRequest(operation.method, operation.path, nil)
				if operation.method == http.MethodPost {
					writer := multipart.NewWriter(&body)
					part, err := writer.CreateFormFile("file", "new.png")
					require.NoError(t, err)
					_, err = part.Write(content)
					require.NoError(t, err)
					require.NoError(t, writer.Close())
					request = httptest.NewRequest(operation.method, operation.path, &body)
					request.Header.Set("Content-Type", writer.FormDataContentType())
				}
				request.Header.Set("New-Api-User", "42")
				for _, sessionCookie := range cookies {
					request.AddCookie(sessionCookie)
				}
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, request)
				assert.Equal(t, identity.status, response.Code, operation.method)
				var result struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
				assert.False(t, result.Success, operation.method)
				assert.NotEmpty(t, result.Message, operation.method)
			}
			remaining, found := model.GetLogoImage("existing.png")
			require.True(t, found)
			assert.Equal(t, content, remaining.Content)
			_, found = model.GetLogoImage("new.png")
			assert.False(t, found)
		})
	}
}

func TestLogoImagePublicRouterWithFrontendBaseURL(t *testing.T) {
	content := prepareLogoImageRouterStorage(t)
	previousMasterNode := common.IsMasterNode
	common.IsMasterNode = false
	t.Cleanup(func() { common.IsMasterNode = previousMasterNode })

	for _, frontendURL := range []string{"", "https://frontend.example.com"} {
		t.Run("frontend="+frontendURL, func(t *testing.T) {
			t.Setenv("FRONTEND_BASE_URL", frontendURL)
			engine := gin.New()
			SetRouter(engine, WebAssets{
				BuildFS: fstest.MapFS{
					"web/default/dist/index.html": {Data: []byte("frontend-shell")},
				},
				IndexPage: []byte("frontend-shell"),
			})

			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, httptest.NewRequest(method, "/logo/existing.png", nil))
				assert.Equal(t, http.StatusOK, response.Code, method)
				assert.Empty(t, response.Header().Get("Location"), method)
				assert.Equal(t, "image/png", response.Header().Get("Content-Type"), method)
				assert.Equal(t, "no-store", response.Header().Get("Cache-Control"), method)
				if method == http.MethodGet {
					assert.Equal(t, content, response.Body.Bytes())
				} else {
					assert.Empty(t, response.Body.Bytes())
				}

				response = httptest.NewRecorder()
				engine.ServeHTTP(response, httptest.NewRequest(method, "/logo/missing.png", nil))
				assert.Equal(t, http.StatusNotFound, response.Code, method)
				assert.Empty(t, response.Header().Get("Location"), method)
				assert.NotContains(t, response.Header().Get("Content-Type"), "text/html", method)
			}

			response := httptest.NewRecorder()
			engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard/example", nil))
			if frontendURL == "" {
				assert.Equal(t, http.StatusOK, response.Code)
				assert.Equal(t, "frontend-shell", response.Body.String())
			} else {
				assert.Equal(t, http.StatusMovedPermanently, response.Code)
				assert.Equal(t, frontendURL+"/dashboard/example", response.Header().Get("Location"))
			}
		})
	}
}
