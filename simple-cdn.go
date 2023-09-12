package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Get IP from env
	ip := os.Getenv("IP_ADDRESS")
	if ip == "" {
		ip = "0.0.0.0"
	}

	// Get port from env
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "3003"
	}

	var router *gin.Engine = gin.Default()

	// Serve static files
	router.Static("/", "./cdn")

	router.Run(ip + ":" + port)
}
