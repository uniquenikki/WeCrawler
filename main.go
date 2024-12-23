package main

import (
	"net/http"
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
	r.GET("/home", func(c *gin.Context) {
		HomeHandler(c)
	})

	// Start the server on port 8080
	if err := r.Run(":8080"); err != nil {
		panic("Failed to start the server: " + err.Error())
	}
}
func HomeHandler(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": "Hi there. This is the Autotron Core API built in Golang!",
	})
}
