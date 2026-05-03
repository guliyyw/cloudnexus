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
		c.JSON(http.StatusOK, response.OK("user-file-svc healthy"))
	})

	log.Println("user-file-svc starting on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("failed to start: %v", err)
	}
}
