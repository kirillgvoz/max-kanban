package handlers

import (
	"strconv"

	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

func IssueWebSocketTicket(c *gin.Context) {
	var request struct {
		OrgID int64 `json:"org_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "org_id is required"})
		return
	}
	if !requireOrgRole(c, int64Param(request.OrgID), roleMember) {
		return
	}
	ticket := ws.DefaultHub.IssueTicket(getUserID(c), request.OrgID)
	if ticket == "" {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create websocket ticket"})
		return
	}
	c.JSON(200, gin.H{"ticket": ticket})
}

func int64Param(value int64) string {
	return strconv.FormatInt(value, 10)
}
