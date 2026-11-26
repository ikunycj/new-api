package controller

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp"
)

const (
	maxLogoImageBytes  = 5 << 20
	maxLogoImagePixels = 16 << 20
)

type logoImage struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func ListLogoImages(c *gin.Context) {
	names := model.ListLogoImageNames()
	images := make([]logoImage, 0, len(names))
	for _, name := range names {
		images = append(images, logoImage{Name: name, URL: "/logo/" + url.PathEscape(name)})
	}
	common.ApiSuccess(c, images)
}

func UploadLogoImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLogoImageBytes+(64<<10))
	fileHeader, err := c.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			common.ApiErrorMsg(c, "图片大小不能超过 5 MiB")
		} else {
			common.ApiErrorMsg(c, "请选择要上传的图片，使用 file 字段提交完整文件")
		}
		return
	}

	// FormFile normalizes filename paths, so validate the original multipart name.
	_, parameters, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Disposition"))
	name := parameters["filename"]
	if err != nil || !validLogoImageName(name) {
		common.ApiErrorMsg(c, "图片文件名无效，请勿包含路径、控制字符或以点开头")
		return
	}
	format, _ := logoImageFormat(name)
	if format == "" {
		common.ApiErrorMsg(c, "仅支持 PNG、JPG、JPEG、WebP 和 GIF 图片")
		return
	}
	if fileHeader.Size > maxLogoImageBytes {
		common.ApiErrorMsg(c, "图片大小不能超过 5 MiB")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		common.ApiErrorMsg(c, "读取上传图片失败，请重新上传")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxLogoImageBytes+1))
	if err != nil {
		common.ApiErrorMsg(c, "读取上传图片失败，请重新上传")
		return
	}
	if len(content) > maxLogoImageBytes {
		common.ApiErrorMsg(c, "图片大小不能超过 5 MiB")
		return
	}
	config, detectedFormat, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || detectedFormat != format {
		common.ApiErrorMsg(c, "图片内容无效，或实际格式与文件扩展名不一致")
		return
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxLogoImagePixels/config.Height {
		common.ApiErrorMsg(c, "图片像素总数不能超过 16777216")
		return
	}
	if _, _, err := image.Decode(bytes.NewReader(content)); err != nil {
		common.ApiErrorMsg(c, "图片内容不完整或无法解码，请重新选择图片")
		return
	}

	if err := model.AddLogoImage(name, content); err != nil {
		if errors.Is(err, model.ErrLogoImageExists) || errors.Is(err, model.ErrLogoImageLimit) {
			common.ApiErrorMsg(c, err.Error())
		} else {
			common.SysError("保存徽标图片失败: " + err.Error())
			common.ApiErrorMsg(c, "保存图片失败，请重试")
		}
		return
	}
	common.ApiSuccess(c, logoImage{Name: name, URL: "/logo/" + url.PathEscape(name)})
}

func DeleteLogoImage(c *gin.Context) {
	name := c.Param("name")
	format, _ := logoImageFormat(name)
	if !validLogoImageName(name) || format == "" {
		common.ApiErrorMsg(c, "图片文件名无效")
		return
	}
	if err := model.DeleteLogoImage(name); err != nil {
		if errors.Is(err, model.ErrLogoImageNotFound) {
			common.ApiErrorMsg(c, err.Error())
		} else {
			common.SysError("删除徽标图片失败: " + err.Error())
			common.ApiErrorMsg(c, "删除图片失败，请刷新列表后重试")
		}
		return
	}
	common.ApiSuccess(c, nil)
}

func ServeLogoImage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	name := c.Param("name")
	_, contentType := logoImageFormat(name)
	if !validLogoImageName(name) || contentType == "" {
		c.Status(http.StatusNotFound)
		return
	}
	image, found := model.GetLogoImage(name)
	if !found {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, name, image.UploadedAt, bytes.NewReader(image.Content))
}

func validLogoImageName(name string) bool {
	if name == "" || len(name) > 255 || !utf8.ValidString(name) || strings.TrimSpace(name) == "" ||
		strings.HasPrefix(name, ".") || strings.ContainsAny(name, `/\`) {
		return false
	}
	for _, character := range name {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func logoImageFormat(name string) (string, string) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "png", "image/png"
	case ".jpg", ".jpeg":
		return "jpeg", "image/jpeg"
	case ".webp":
		return "webp", "image/webp"
	case ".gif":
		return "gif", "image/gif"
	default:
		return "", ""
	}
}
