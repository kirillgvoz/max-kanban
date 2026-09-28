package handlers

import (
	"strconv"
	"strings"

	"kanban/db"
	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct{}

func NewCommentHandler() *CommentHandler { return &CommentHandler{} }

func (h *CommentHandler) List(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, task_id, user_id, username, display_name, text, created_at
		FROM comments WHERE task_id = $1 ORDER BY created_at, id
	`, taskID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load comments"})
		return
	}
	defer rows.Close()
	comments := make([]models.Comment, 0)
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.TaskID, &comment.UserID, &comment.Username, &comment.DisplayName, &comment.Text, &comment.CreatedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load comments"})
			return
		}
		comments = append(comments, comment)
	}
	c.JSON(200, comments)
}

func (h *CommentHandler) Create(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.CommentCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Comment text is required"})
		return
	}
	request.Text = strings.TrimSpace(request.Text)
	if request.Text == "" {
		c.JSON(400, models.ErrorResponse{Error: "Comment text is required"})
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	var comment models.Comment
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO comments (task_id, user_id, username, display_name, text)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, task_id, user_id, username, display_name, text, created_at
	`, taskID, getUserID(c), getUsername(c), getDisplayName(c), request.Text).Scan(
		&comment.ID, &comment.TaskID, &comment.UserID, &comment.Username, &comment.DisplayName, &comment.Text, &comment.CreatedAt,
	)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create comment"})
		return
	}
	orgID, _ := taskOrganization(ctx, c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "comment:created", comment)
	c.JSON(201, comment)
}

func (h *CommentHandler) Delete(c *gin.Context) {
	taskID, allowed := requireCommentAccess(c, c.Param("id"))
	if !allowed {
		return
	}
	commentID, _ := positiveID(c.Param("id"))
	orgID, _ := taskOrganization(c.Request.Context(), strconv.FormatInt(taskID, 10))
	result, err := db.Pool.Exec(c.Request.Context(), `DELETE FROM comments WHERE id = $1`, commentID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete comment"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Comment not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "comment:deleted", gin.H{"id": commentID, "task_id": taskID})
	c.JSON(200, gin.H{"ok": true})
}
