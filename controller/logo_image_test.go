package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type logoImageTestResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func newLogoImageTestRouter() *gin.Engine {
	router := gin.New()
	router.GET("/api/option/logo-images", ListLogoImages)
	router.POST("/api/option/logo-images", UploadLogoImage)
	router.DELETE("/api/option/logo-images/:name", DeleteLogoImage)
	router.GET("/logo/:name", ServeLogoImage)
	router.HEAD("/logo/:name", ServeLogoImage)
	return router
}

func useLogoImageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run PostgreSQL logo image tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { assert.NoError(t, pool.Close()) })
	schema := fmt.Sprintf("logo_image_test_%d", time.Now().UnixNano())
	require.NoError(t, db.Exec(`CREATE SCHEMA "`+schema+`"`).Error)
	t.Cleanup(func() { assert.NoError(t, db.Exec(`DROP SCHEMA "`+schema+`" CASCADE`).Error) })
	require.NoError(t, db.Exec(`SET search_path TO "`+schema+`"`).Error)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	previousDB, previousType := model.DB, common.MainDatabaseType()
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	t.Cleanup(func() {
		for _, name := range model.ListLogoImageNames() {
			assert.NoError(t, model.DeleteLogoImage(name))
		}
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	return db
}

func newLogoImageUploadRequest(t *testing.T, name string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/api/option/logo-images", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func logoImageTestPNG(t *testing.T, pixel color.RGBA) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	picture.SetRGBA(0, 0, pixel)
	var content bytes.Buffer
	require.NoError(t, png.Encode(&content, picture))
	return content.Bytes()
}

func TestLogoImageLifecycle(t *testing.T) {
	db := useLogoImageTestDB(t)
	router := newLogoImageTestRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/option/logo-images", nil))
	var list logoImageTestResponse[[]logoImage]
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &list))
	require.True(t, list.Success)
	assert.Equal(t, []logoImage{}, list.Data)

	name := "品牌 徽标.png"
	content := logoImageTestPNG(t, color.RGBA{R: 255, A: 255})
	response = httptest.NewRecorder()
	router.ServeHTTP(response, newLogoImageUploadRequest(t, name, content))
	var upload logoImageTestResponse[logoImage]
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &upload))
	require.True(t, upload.Success, upload.Message)
	assert.Equal(t, name, upload.Data.Name)
	assert.Equal(t, "/logo/"+url.PathEscape(name), upload.Data.URL)
	var saved model.Option
	require.NoError(t, db.First(&saved, `"key" = ?`, "LogoImages").Error)
	require.Contains(t, saved.Value, base64.StdEncoding.EncodeToString(content))

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/option/logo-images", nil))
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &list))
	require.True(t, list.Success)
	assert.Equal(t, []logoImage{upload.Data}, list.Data)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, upload.Data.URL, nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "image/png", response.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.Equal(t, content, response.Body.Bytes())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodHead, upload.Data.URL, nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, strconv.Itoa(len(content)), response.Header().Get("Content-Length"))
	assert.Empty(t, response.Body.Bytes())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/option/logo-images/"+url.PathEscape(name), nil))
	var deletion logoImageTestResponse[any]
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &deletion))
	require.True(t, deletion.Success, deletion.Message)
	require.NoError(t, db.First(&saved, `"key" = ?`, "LogoImages").Error)
	assert.JSONEq(t, "{}", saved.Value)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, upload.Data.URL, nil))
	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/option/logo-images", nil))
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &list))
	assert.True(t, list.Success)
	assert.Empty(t, list.Data)
}

func TestLogoImageAcceptsSupportedRasterFormats(t *testing.T) {
	useLogoImageTestDB(t)
	router := newLogoImageTestRouter()
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	picture.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	var jpegContent bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpegContent, picture, nil))
	var gifContent bytes.Buffer
	require.NoError(t, gif.Encode(&gifContent, picture, nil))
	webpContent, err := base64.StdEncoding.DecodeString("UklGRhwAAABXRUJQVlA4TA8AAAAvAAAAAAcQ/Y/+ByKi/wEA")
	require.NoError(t, err)

	for _, test := range []struct {
		name        string
		content     []byte
		contentType string
	}{
		{"image.PNG", logoImageTestPNG(t, color.RGBA{A: 255}), "image/png"},
		{"image.jpg", jpegContent.Bytes(), "image/jpeg"},
		{"image.jpeg", jpegContent.Bytes(), "image/jpeg"},
		{"image.gif", gifContent.Bytes(), "image/gif"},
		{"image.webp", webpContent, "image/webp"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, newLogoImageUploadRequest(t, test.name, test.content))
			var upload logoImageTestResponse[logoImage]
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &upload))
			require.True(t, upload.Success, upload.Message)
			response = httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, upload.Data.URL, nil))
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, test.contentType, response.Header().Get("Content-Type"))
			assert.Equal(t, test.content, response.Body.Bytes())
		})
	}
}

func TestLogoImageRejectsInvalidUploads(t *testing.T) {
	router := newLogoImageTestRouter()
	pngContent := logoImageTestPNG(t, color.RGBA{A: 255})
	oversizedDimensions := bytes.Clone(pngContent)
	binary.BigEndian.PutUint32(oversizedDimensions[16:20], 65535)
	binary.BigEndian.PutUint32(oversizedDimensions[20:24], 65535)
	binary.BigEndian.PutUint32(oversizedDimensions[29:33], crc32.ChecksumIEEE(oversizedDimensions[12:29]))

	for _, test := range []struct {
		name    string
		content []byte
	}{
		{"../escape.png", pngContent},
		{`..\escape.png`, pngContent},
		{".hidden.png", pngContent},
		{"control\x01.png", pngContent},
		{"image.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"image.ico", pngContent},
		{"image.jpg", pngContent},
		{"empty.png", nil},
		{"invalid.png", []byte("not an image")},
		{"truncated.png", pngContent[:len(pngContent)-15]},
		{"large.png", bytes.Repeat([]byte{0}, (5<<20)+1)},
		{"dimensions.png", oversizedDimensions},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, newLogoImageUploadRequest(t, test.name, test.content))
			assert.Equal(t, http.StatusOK, response.Code)
			var upload logoImageTestResponse[logoImage]
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &upload))
			assert.False(t, upload.Success)
			assert.NotEmpty(t, upload.Message)
		})
	}
}

func TestLogoImageConcurrentDuplicatesPreserveSuccessfulUpload(t *testing.T) {
	useLogoImageTestDB(t)
	router := newLogoImageTestRouter()
	contents := [][]byte{
		logoImageTestPNG(t, color.RGBA{R: 255, A: 255}),
		logoImageTestPNG(t, color.RGBA{B: 255, A: 255}),
	}
	requests := []*http.Request{
		newLogoImageUploadRequest(t, "same.png", contents[0]),
		newLogoImageUploadRequest(t, "same.png", contents[1]),
	}
	responses := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
	var workers sync.WaitGroup
	for index := range requests {
		workers.Add(1)
		go func() {
			defer workers.Done()
			router.ServeHTTP(responses[index], requests[index])
		}()
	}
	workers.Wait()

	successes := 0
	saved, exists := model.GetLogoImage("same.png")
	require.True(t, exists)
	for index, response := range responses {
		var upload logoImageTestResponse[logoImage]
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &upload))
		if upload.Success {
			successes++
			assert.Equal(t, contents[index], saved.Content)
		} else {
			assert.Contains(t, upload.Message, "同名")
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, []string{"same.png"}, model.ListLogoImageNames())
}

func TestLogoImageConcurrentUploadsRespectLimit(t *testing.T) {
	useLogoImageTestDB(t)
	router := newLogoImageTestRouter()
	content := logoImageTestPNG(t, color.RGBA{A: 255})
	for index := range 9 {
		require.NoError(t, model.AddLogoImage(fmt.Sprintf("logo-%d.png", index), content))
	}
	requests := []*http.Request{
		newLogoImageUploadRequest(t, "last-a.png", content),
		newLogoImageUploadRequest(t, "last-b.png", content),
	}
	responses := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
	var workers sync.WaitGroup
	for index := range requests {
		workers.Go(func() { router.ServeHTTP(responses[index], requests[index]) })
	}
	workers.Wait()
	successes := 0
	for _, response := range responses {
		var result logoImageTestResponse[logoImage]
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
		if result.Success {
			successes++
		} else {
			assert.Equal(t, model.ErrLogoImageLimit.Error(), result.Message)
		}
	}
	assert.Equal(t, 1, successes)
	assert.Len(t, model.ListLogoImageNames(), 10)
	require.NoError(t, model.DeleteLogoImage("logo-0.png"))
	require.NoError(t, model.AddLogoImage("replacement.png", content))
	assert.Len(t, model.ListLogoImageNames(), 10)
}

func TestLogoImageReportsStorageFailures(t *testing.T) {
	db := useLogoImageTestDB(t)
	require.NoError(t, db.Exec(`ALTER TABLE options ADD CHECK (value = '{}')`).Error)
	router := newLogoImageTestRouter()
	request := newLogoImageUploadRequest(t, "image.png", logoImageTestPNG(t, color.RGBA{A: 255}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	var result logoImageTestResponse[any]
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	assert.False(t, result.Success)
	assert.Equal(t, "保存图片失败，请重试", result.Message)
	assert.Empty(t, model.ListLogoImageNames())
}
