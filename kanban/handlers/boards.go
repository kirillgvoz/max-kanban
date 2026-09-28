package handlers

import (
	"strings"

	"kanban/db"
	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

type BoardHandler struct{}

func NewBoardHandler() *BoardHandler { return &BoardHandler{} }

func (h *BoardHandler) ListByOrg(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleMember) {
		return
	}
	orgID, _ := positiveID(c.Param("id"))
	includeArchived := c.Query("include_archived") == "true"
	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, org_id, name, description, is_archived, created_by, created_at
		FROM boards
		WHERE org_id = $1 AND (is_archived = FALSE OR $2)
		ORDER BY is_archived, created_at DESC
	`, orgID, includeArchived)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load boards"})
		return
	}
	defer rows.Close()

	boards := make([]models.Board, 0)
	for rows.Next() {
		var board models.Board
		if err := rows.Scan(&board.ID, &board.OrgID, &board.Name, &board.Description, &board.IsArchived, &board.CreatedBy, &board.CreatedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load boards"})
			return
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load boards"})
		return
	}
	c.JSON(200, boards)
}

func (h *BoardHandler) Create(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleAdmin) {
		return
	}
	var request models.BoardCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid board data"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		c.JSON(400, models.ErrorResponse{Error: "Board name is required"})
		return
	}
	columns := request.Columns
	if len(columns) == 0 {
		columns = []models.ColumnCreate{
			{Name: "К выполнению", Color: "#6366F1"},
			{Name: "В работе", Color: "#F59E0B"},
			{Name: "Готово", Color: "#10B981"},
		}
	}
	if len(columns) > 30 {
		c.JSON(400, models.ErrorResponse{Error: "A board can have at most 30 statuses"})
		return
	}
	for index := range columns {
		columns[index].Name = strings.TrimSpace(columns[index].Name)
		if columns[index].Name == "" || len(columns[index].Name) > 80 {
			c.JSON(400, models.ErrorResponse{Error: "Status names must contain 1-80 characters"})
			return
		}
		if columns[index].Color == "" {
			columns[index].Color = "#6B7280"
		}
		if !validHexColor(columns[index].Color) {
			c.JSON(400, models.ErrorResponse{Error: "Status colors must use #RRGGBB"})
			return
		}
	}

	orgID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create board"})
		return
	}
	defer tx.Rollback(ctx)

	var board models.Board
	err = tx.QueryRow(ctx, `
		INSERT INTO boards (org_id, name, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, org_id, name, description, is_archived, created_by, created_at
	`, orgID, request.Name, request.Description, getUserID(c)).Scan(&board.ID, &board.OrgID, &board.Name, &board.Description, &board.IsArchived, &board.CreatedBy, &board.CreatedAt)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create board"})
		return
	}
	for position, column := range columns {
		if _, err = tx.Exec(ctx, `
			INSERT INTO columns (board_id, name, position, color)
			VALUES ($1, $2, $3, $4)
		`, board.ID, column.Name, position, column.Color); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to create board"})
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create board"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "board:created", board)
	c.JSON(201, board)
}

func (h *BoardHandler) Get(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleMember) {
		return
	}
	boardID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()

	var board models.Board
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, is_archived, created_by, created_at
		FROM boards WHERE id = $1 AND is_archived = FALSE
	`, boardID).Scan(&board.ID, &board.OrgID, &board.Name, &board.Description, &board.IsArchived, &board.CreatedBy, &board.CreatedAt)
	if err != nil {
		respondMissing(c, err, "Board")
		return
	}

	columnRows, err := db.Pool.Query(ctx, `
		SELECT id, board_id, name, position, color
		FROM columns WHERE board_id = $1 ORDER BY position, id
	`, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load board"})
		return
	}
	defer columnRows.Close()
	columns := make([]models.Column, 0)
	for columnRows.Next() {
		var column models.Column
		if err := columnRows.Scan(&column.ID, &column.BoardID, &column.Name, &column.Position, &column.Color); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load board"})
			return
		}
		columns = append(columns, column)
	}

	memberRows, err := db.Pool.Query(ctx, `
		SELECT org_id, user_id, username, display_name, role, joined_at
		FROM org_members WHERE org_id = $1 ORDER BY joined_at
	`, board.OrgID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load board"})
		return
	}
	defer memberRows.Close()
	members := make([]models.OrgMember, 0)
	for memberRows.Next() {
		var member models.OrgMember
		if err := memberRows.Scan(&member.OrgID, &member.UserID, &member.Username, &member.DisplayName, &member.Role, &member.JoinedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load board"})
			return
		}
		members = append(members, member)
	}
	c.JSON(200, models.BoardDetail{Board: board, Columns: columns, Members: members})
}

func (h *BoardHandler) Update(c *gin.Context) {
	if !requireBoardRoleWithArchived(c, c.Param("id"), roleAdmin, true) {
		return
	}
	var request models.BoardUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid board data"})
		return
	}
	if request.Name != nil {
		trimmed := strings.TrimSpace(*request.Name)
		request.Name = &trimmed
	}
	boardID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `
		UPDATE boards
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    is_archived = COALESCE($3, is_archived)
		WHERE id = $4
	`, request.Name, request.Description, request.Archived, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update board"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Board not found"})
		return
	}
	orgID, _ := boardOrganization(c.Request.Context(), c.Param("id"), true)
	ws.BroadcastToOrgDirect(orgID, "board:updated", gin.H{"id": boardID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *BoardHandler) Delete(c *gin.Context) {
	if !requireBoardRoleWithArchived(c, c.Param("id"), roleAdmin, true) {
		return
	}
	boardID, _ := positiveID(c.Param("id"))
	orgID, err := boardOrganization(c.Request.Context(), c.Param("id"), true)
	if err != nil {
		respondMissing(c, err, "Board")
		return
	}
	result, err := db.Pool.Exec(c.Request.Context(), `UPDATE boards SET is_archived = TRUE WHERE id = $1`, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to archive board"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Board not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "board:deleted", gin.H{"id": boardID})
	c.JSON(200, gin.H{"ok": true})
}
