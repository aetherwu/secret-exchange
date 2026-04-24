package routers

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadToken returns an upload token — stubbed until file storage is configured.
func UploadToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "", "errMsg": "file upload not configured"})
}

type UploadCallbackModel struct {
	Key    string `json:"key"`
	Hash   string `json:"hash"`
	Fsize  int    `json:"fsize"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

func UploadCallback(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusOK, DefaultError())
}

func UploadImage(imageData []byte, format string) (string, error) {
	return "", fmt.Errorf("file upload not configured")
}

func DownloadImage(url string) (image.Image, string, error) {
	client := &http.Client{
		Timeout: time.Duration(20 * time.Second),
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	img, f, err := image.Decode(res.Body)
	return img, f, err
}

func UploadWXAvatar(avatar image.Image, format string) (string, error) {
	return "", fmt.Errorf("file upload not configured")
}
