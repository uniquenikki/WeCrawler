package main

import (
	"webCrawler/webcrawler"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(gin.Recovery())

	// Define a GET endpoint
	r.GET("/", func(c *gin.Context) {
		webcrawler.WebCrawler(c)
	})

	// Start the server on port 8080
	if err := r.Run(":8080"); err != nil {
		panic("Failed to start the server: " + err.Error())
	}
}
