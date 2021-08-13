package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"strings"
)

func uploadFile(c *gin.Context) () {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": err.Error(),
		})
		return
	}
	savePath := c.Query("path")
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err": nil,
	})
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func downFile(c *gin.Context) () {
	file := c.Query("file")
	if !pathExists(file) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "file not found",
		})
		return
	}
	tmp := strings.Split(file, "/")
	var filename string
	if len(tmp) > 0 {
		filename = tmp[len(tmp)-1]
	}
	c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.File(file)
}

func main() {
	flag.Parse()
	args := flag.Args()
	router := gin.Default()
	router.POST("/uploadFile", uploadFile)
	router.GET("/downFile", downFile)
	log.Fatal(router.Run(":" + args[0]))
}
