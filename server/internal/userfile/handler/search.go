package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	userRepo *repository.UserRepository
}

func NewSearchHandler(userRepo *repository.UserRepository) *SearchHandler {
	return &SearchHandler{userRepo: userRepo}
}

func (h *SearchHandler) HandleSearch(c *gin.Context) {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.userRepo.SearchUsers(q, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "搜索失败"))
		return
	}
	if users == nil {
		users = []model.UserBrief{}
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}
