package main

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func serve(useSystemYoutubeDL bool) error {
	r := gin.Default()
	r.POST("/eps/re-fetch", func(c *gin.Context) {
		var json struct {
			Key string `json:"key"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		value, ok := os.LookupEnv("AKS_KEYS")
		if ok {
			allowedKeys := strings.Split(value, ",")
			found := false
			for _, allowedKey := range allowedKeys {
				if len(allowedKey) < 3 {
					continue
				}
				if allowedKey == json.Key {
					found = true
					break
				}
			}
			if !found {
				c.JSON(400, gin.H{"error": "Invalid key"})
				return
			}
		}

		downloadingLock.Lock()
		if downloading {
			downloadingLock.Unlock()
			c.JSON(400, gin.H{"error": "Buzzy updating"})
			return
		}
		downloading = true
		downloadingLock.Unlock()

		defer func() {
			downloadingLock.Lock()
			downloading = false
			downloadingLock.Unlock()
		}()

		err := downloadLatestVideosMeta(useSystemYoutubeDL)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err = checkDownloadedVideos()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"ok": true})
	})

	r.GET("/eps", func(c *gin.Context) {
		videosLock.Lock()
		defer videosLock.Unlock()
		c.JSON(200, videos)
	})

	corsConf := cors.DefaultConfig()
	corsConf.AllowAllOrigins = true
	corsConf.AllowCredentials = true
	corsConf.AllowHeaders = nil
	r.Use(cors.New(corsConf))

	return r.Run("0.0.0.0:9090")
}
