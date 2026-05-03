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
		c.JSON(http.StatusOK, response.OK("im-svc healthy"))
	})

	log.Println("im-svc starting on :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("failed to start: %v", err)
	}
}
