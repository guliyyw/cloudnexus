package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/cloudnexus/server/pkg/response"
)

func main() {
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OK("docker-svc healthy"))
	})

	log.Println("docker-svc starting on :8083")
	if err := r.Run(":8083"); err != nil {
		log.Fatalf("failed to start: %v", err)
	}
}
