package controller

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	backgroundUploadDir      = "uploads/backgrounds"
	backgroundMaxUploadBytes = 8 << 20
)

var backgroundContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func UploadCustomBackground(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请选择背景图片文件"})
		return
	}
	if file.Size <= 0 || file.Size > backgroundMaxUploadBytes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "图片大小必须在 8 MB 以内"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "读取上传文件失败"})
		return
	}
	defer src.Close()

	header := make([]byte, 512)
	n, readErr := io.ReadFull(src, header)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "读取上传文件失败"})
		return
	}
	contentType := http.DetectContentType(header[:n])
	ext, ok := backgroundContentTypes[contentType]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅支持 jpg/png/webp 图片"})
		return
	}
	if _, err = src.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "读取上传文件失败"})
		return
	}

	if err = os.MkdirAll(backgroundUploadDir, 0o755); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建上传目录失败"})
		return
	}
	filename, err := newBackgroundFilename(ext)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "生成文件名失败"})
		return
	}
	if err = saveBackgroundAtomically(src, filename); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "上传失败"})
		return
	}

	url := "/api/public/backgrounds/" + filename
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"url": url}})
}

func newBackgroundFilename(ext string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "bg_" + hex.EncodeToString(random) + ext, nil
}

func saveBackgroundAtomically(src io.Reader, filename string) (err error) {
	tmp, err := os.CreateTemp(backgroundUploadDir, ".background-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	written, err := io.Copy(tmp, io.LimitReader(src, backgroundMaxUploadBytes+1))
	if err != nil {
		return err
	}
	if written <= 0 || written > backgroundMaxUploadBytes {
		return errors.New("invalid background image size")
	}
	if err = tmp.Chmod(0o644); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(backgroundUploadDir, filename))
}

func GetPublicBackground(c *gin.Context) {
	filename := c.Param("filename")
	if !validBackgroundFilename(filename) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件不存在"})
		return
	}
	filePath := filepath.Join(backgroundUploadDir, filename)
	info, err := os.Lstat(filePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件不存在"})
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(filePath)
}

func validBackgroundFilename(filename string) bool {
	if filename == "" || filename != filepath.Base(filename) || strings.ContainsAny(filename, `/\\`) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(filename))
	return strings.HasPrefix(filename, "bg_") && (ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp")
}
