package handlers

import (
	"kanban/middleware"
	"kanban/models"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	BotToken string
}

func NewAuthHandler(botToken string) *AuthHandler {
	return &AuthHandler{BotToken: botToken}
}

func (h *AuthHandler) Validate(c *gin.Context) {
	var req models.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid request"})
		return
	}

	if err := middleware.ValidateInitData(req.InitData, h.BotToken); err != nil {
		c.JSON(401, models.AuthResponse{OK: false, Error: err.Error()})
		return
	}

	user, err := middleware.ParseUser(req.InitData)
	if err != nil {
		c.JSON(400, models.AuthResponse{OK: false, Error: err.Error()})
		return
	}

	c.JSON(200, models.AuthResponse{OK: true, User: *user})
}
