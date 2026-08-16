package router

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

var publicPageLocales = []string{"en", "zhCN", "zhTW", "fr", "ja", "ru", "vi"}

const (
	documentationLocale = "zhCN"
	publicHomeFile      = "home.html"
)

var publicPageLanguageMatcher = language.NewMatcher([]language.Tag{
	language.English,
	language.SimplifiedChinese,
	language.TraditionalChinese,
	language.French,
	language.Japanese,
	language.Russian,
	language.Vietnamese,
})

type WebAssets struct {
	BuildFS   fs.FS
	IndexPage []byte
}

type docsManifest struct {
	Locales map[string]map[string]string `json:"locales"`
	Version int                          `json:"version"`
}

func contentETag(data []byte) string {
	digest := sha256.Sum256(data)
	return `"` + hex.EncodeToString(digest[:]) + `"`
}

func matchesETag(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

func SetWebRouter(router *gin.Engine, assets WebAssets) {
	webFS := common.EmbedFolder(assets.BuildFS, "web/default/dist")
	manifest := docsManifest{}
	manifestETag := ""
	if manifestJSON, err := fs.ReadFile(assets.BuildFS, "web/default/dist/static/docs/manifest.json"); err == nil {
		manifestETag = contentETag(manifestJSON)
		_ = common.Unmarshal(manifestJSON, &manifest)
	}
	documentationRoutes := manifest.Locales[documentationLocale]
	publicPageRoutes := make(map[string]struct{}, len(documentationRoutes)+1)
	publicPageRoutes["/"] = struct{}{}
	prerenderPages := make(map[string][]byte, len(publicPageLocales)+len(documentationRoutes))
	for _, locale := range publicPageLocales {
		page, err := fs.ReadFile(
			assets.BuildFS,
			path.Join("web/default/dist/prerender", locale, publicHomeFile),
		)
		if err == nil {
			prerenderPages[locale+":/"] = page
		}
	}
	for routePath := range documentationRoutes {
		fileName, ok := documentationPrerenderFile(routePath)
		if !ok {
			continue
		}
		publicPageRoutes[routePath] = struct{}{}
		page, err := fs.ReadFile(
			assets.BuildFS,
			path.Join("web/default/dist/prerender", documentationLocale, fileName),
		)
		if err == nil {
			prerenderPages[documentationLocale+":"+routePath] = page
		}
	}

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path != "/static/docs/manifest.json" || manifestETag == "" {
			c.Next()
			return
		}

		c.Header("ETag", manifestETag)
		if matchesETag(c.GetHeader("If-None-Match"), manifestETag) {
			c.AbortWithStatus(http.StatusNotModified)
			return
		}
		c.Next()
	})
	router.Use(static.Serve("/", webFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		requestPath := c.Request.URL.Path
		if requestPath == "/static" || strings.HasPrefix(requestPath, "/static/") || requestPath == "/assets" || strings.HasPrefix(requestPath, "/assets/") || requestPath == "/prerender" || strings.HasPrefix(requestPath, "/prerender/") {
			c.Status(http.StatusNotFound)
			return
		}
		if strings.HasPrefix(requestPath, "/v1") || strings.HasPrefix(requestPath, "/api") {
			controller.RelayNotFound(c)
			return
		}

		canonicalPath := strings.TrimSuffix(requestPath, "/")
		if canonicalPath == "" {
			canonicalPath = "/"
		}
		if canonicalPath == "/docs/ai-model" {
			c.Redirect(http.StatusTemporaryRedirect, "/docs/api/integration")
			return
		}
		if _, isPublicPage := publicPageRoutes[canonicalPath]; isPublicPage {
			if !constant.Setup {
				c.Redirect(http.StatusTemporaryRedirect, "/setup")
				return
			}

			common.OptionMapRWMutex.RLock()
			homePageContent := common.OptionMap["HomePageContent"]
			headerNavModules := common.OptionMap["HeaderNavModules"]
			common.OptionMapRWMutex.RUnlock()
			if canonicalPath == "/" && strings.TrimSpace(homePageContent) != "" {
				c.Header("Cache-Control", "no-cache, must-revalidate")
				c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
				return
			}

			locale := selectPublicPageLocale(c)
			if strings.HasPrefix(canonicalPath, "/docs") {
				locale = documentationLocale
			}
			if template, ok := prerenderPages[locale+":"+canonicalPath]; ok {
				logo := strings.TrimSpace(common.Logo)
				if logo == "" || logo == "/logo.png" {
					logo = "/logo-56.webp"
				}
				serverAddress := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
				if serverAddress == "" {
					scheme := "http"
					if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
						scheme = "https"
					}
					serverAddress = scheme + "://" + c.Request.Host
				}

				bootstrap := map[string]any{
					"home_page_content":        "",
					"home_page_content_loaded": canonicalPath == "/",
					"locale":                   locale,
					"setup":                    true,
					"status": map[string]any{
						"HeaderNavModules": headerNavModules,
						"logo":             logo,
						"server_address":   serverAddress,
						"system_name":      common.SystemName,
					},
				}
				if fileName := manifest.Locales[locale][canonicalPath]; fileName != "" {
					bootstrap["docs"] = map[string]string{
						"file_name": fileName,
						"route":     canonicalPath,
					}
				}
				bootstrapJSON, err := common.Marshal(bootstrap)
				if err == nil {
					pageHTML := string(template)
					pageHTML = strings.ReplaceAll(pageHTML, "__NEW_API_SYSTEM_NAME__", html.EscapeString(common.SystemName))
					pageHTML = strings.ReplaceAll(pageHTML, "__NEW_API_LOGO__", html.EscapeString(logo))
					pageHTML = strings.ReplaceAll(pageHTML, "__NEW_API_SERVER_ADDRESS__", html.EscapeString(serverAddress))
					pageHTML = strings.Replace(pageHTML, "<title>All Token API</title>", "<title>"+html.EscapeString(common.SystemName)+"</title>", 1)
					pageHTML = strings.Replace(pageHTML, `<meta name="title" content="All Token API" />`, `<meta name="title" content="`+html.EscapeString(common.SystemName)+`" />`, 1)
					pageHTML = strings.Replace(pageHTML, "<!--new-api-bootstrap-->", "<script>window.__NEW_API_PUBLIC_BOOTSTRAP__="+string(bootstrapJSON)+"</script>", 1)

					c.Header("Cache-Control", "no-cache, must-revalidate")
					c.Header("Vary", "Cookie, Accept-Language")
					c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(pageHTML))
					return
				}
			}
		}

		c.Header("Cache-Control", "no-cache, must-revalidate")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
}

func documentationPrerenderFile(routePath string) (string, bool) {
	if routePath == "/docs" {
		return "docs/index.html", true
	}
	if !strings.HasPrefix(routePath, "/docs/") || path.Clean(routePath) != routePath || strings.Contains(routePath, "\\") {
		return "", false
	}
	return strings.TrimPrefix(routePath, "/") + ".html", true
}

func selectPublicPageLocale(c *gin.Context) string {
	if cookieValue, err := c.Cookie("i18next"); err == nil {
		if decoded, decodeErr := url.QueryUnescape(cookieValue); decodeErr == nil {
			cookieValue = decoded
		}
		if locale := normalizePublicPageLocale(cookieValue); locale != "" {
			return locale
		}
	}

	_, index := language.MatchStrings(publicPageLanguageMatcher, c.GetHeader("Accept-Language"))
	if index >= 0 && index < len(publicPageLocales) {
		return publicPageLocales[index]
	}
	return "en"
}

func normalizePublicPageLocale(value string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	switch {
	case normalized == "zh-tw", normalized == "zhtw", normalized == "zh-hk", normalized == "zh-mo", strings.HasPrefix(normalized, "zh-hant"):
		return "zhTW"
	case normalized == "zh", normalized == "zh-cn", normalized == "zhcn", strings.HasPrefix(normalized, "zh-hans"):
		return "zhCN"
	case strings.HasPrefix(normalized, "en"):
		return "en"
	case strings.HasPrefix(normalized, "fr"):
		return "fr"
	case strings.HasPrefix(normalized, "ja"):
		return "ja"
	case strings.HasPrefix(normalized, "ru"):
		return "ru"
	case strings.HasPrefix(normalized, "vi"):
		return "vi"
	default:
		return ""
	}
}
