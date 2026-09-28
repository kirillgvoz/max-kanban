package handlers

import (
	"strings"

	"kanban/db"
	"kanban/models"

	"github.com/gin-gonic/gin"
)

type BoardChatHandler struct{}

func NewBoardChatHandler() *BoardChatHandler { return &BoardChatHandler{} }

func (h *BoardChatHandler) List(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleAdmin) {
		return
	}
	boardID, _ := positiveID(c.Param("id"))
	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, board_id, chat_id, title, created_by, created_at
		FROM board_chats WHERE board_id = $1 ORDER BY created_at, id
	`, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load board chats"})
		return
	}
	defer rows.Close()
	chats := make([]models.BoardChat, 0)
	for rows.Next() {
		var chat models.BoardChat
		if err := rows.Scan(&chat.ID, &chat.BoardID, &chat.ChatID, &chat.Title, &chat.CreatedBy, &chat.CreatedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load board chats"})
			return
		}
		chats = append(chats, chat)
	}
	c.JSON(200, chats)
}

func (h *BoardChatHandler) Create(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleAdmin) {
		return
	}
	var request models.BoardChatCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid chat data"})
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	boardID, _ := positiveID(c.Param("id"))
	var chat models.BoardChat
	err := db.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO board_chats (board_id, chat_id, title, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (board_id, chat_id) DO UPDATE SET title = EXCLUDED.title
		RETURNING id, board_id, chat_id, title, created_by, created_at
	`, boardID, request.ChatID, request.Title, getUserID(c)).Scan(&chat.ID, &chat.BoardID, &chat.ChatID, &chat.Title, &chat.CreatedBy, &chat.CreatedAt)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to save board chat"})
		return
	}
	c.JSON(201, chat)
}

func (h *BoardChatHandler) Delete(c *gin.Context) {
	boardChatID, _ := positiveID(c.Param("id"))
	var boardID int64
	if err := db.Pool.QueryRow(c.Request.Context(), `SELECT board_id FROM board_chats WHERE id = $1`, boardChatID).Scan(&boardID); err != nil {
		respondMissing(c, err, "Board chat")
		return
	}
	ctx := c.Request.Context()
	var orgID int64
	if err := db.Pool.QueryRow(ctx, `SELECT org_id FROM boards WHERE id = $1`, boardID).Scan(&orgID); err != nil {
		respondMissing(c, err, "Board")
		return
	}
	if !requireOrgRoleByID(c, orgID, roleAdmin) {
		return
	}
	result, err := db.Pool.Exec(ctx, `DELETE FROM board_chats WHERE id = $1`, boardChatID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete board chat"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Board chat not found"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
