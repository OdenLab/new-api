package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const backgroundUploadDir = "uploads/backgrounds"

func UploadCustomBackground(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请选择背景图片文件"})
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅支持 jpg/png/webp 图片"})
		return
	}
	if err = os.MkdirAll(backgroundUploadDir, 0o755); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "创建上传目录失败"})
		return
	}
	filename := fmt.Sprintf("bg_%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(backgroundUploadDir, filename)
	if err = c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "上传失败"})
		return
	}
	url := "/api/public/backgrounds/" + filename
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"url": url}})
}

func GetPublicBackground(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	if filename == "." || filename == "" {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件不存在"})
		return
	}
	filePath := filepath.Join(backgroundUploadDir, filename)
	c.File(filePath)
}

