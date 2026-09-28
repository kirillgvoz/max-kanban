package handlers

import (
	"encoding/hex"
	"strings"

	"kanban/db"
	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

type ColumnHandler struct{}

func NewColumnHandler() *ColumnHandler { return &ColumnHandler{} }

func (h *ColumnHandler) Create(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleAdmin) {
		return
	}
	var request models.ColumnCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid status data"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		c.JSON(400, models.ErrorResponse{Error: "Status name is required"})
		return
	}
	if request.Color == "" {
		request.Color = "#6B7280"
	}
	if !validHexColor(request.Color) {
		c.JSON(400, models.ErrorResponse{Error: "Status color must use #RRGGBB"})
		return
	}

	boardID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create status"})
		return
	}
	defer tx.Rollback(ctx)

	var count int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM columns WHERE board_id = $1`, boardID).Scan(&count); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create status"})
		return
	}
	if count >= 30 {
		c.JSON(409, models.ErrorResponse{Error: "A board can have at most 30 statuses"})
		return
	}
	position := count
	if request.Position != nil {
		position = *request.Position
		if position > count {
			position = count
		}
	}
	if _, err = tx.Exec(ctx, `
		UPDATE columns SET position = position + 1
		WHERE board_id = $1 AND position >= $2
	`, boardID, position); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create status"})
		return
	}
	var column models.Column
	err = tx.QueryRow(ctx, `
		INSERT INTO columns (board_id, name, position, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id, board_id, name, position, color
	`, boardID, request.Name, position, request.Color).Scan(&column.ID, &column.BoardID, &column.Name, &column.Position, &column.Color)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create status"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create status"})
		return
	}
	orgID, _ := boardOrganization(c.Request.Context(), c.Param("id"), false)
	ws.BroadcastToOrgDirect(orgID, "column:created", column)
	c.JSON(201, column)
}

func (h *ColumnHandler) Update(c *gin.Context) {
	if !requireColumnRole(c, c.Param("id"), roleAdmin) {
		return
	}
	var request models.ColumnUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid status data"})
		return
	}
	if request.Name != nil {
		trimmed := strings.TrimSpace(*request.Name)
		request.Name = &trimmed
	}
	if request.Color != nil && !validHexColor(*request.Color) {
		c.JSON(400, models.ErrorResponse{Error: "Status color must use #RRGGBB"})
		return
	}
	columnID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `
		UPDATE columns SET name = COALESCE($1, name), color = COALESCE($2, color)
		WHERE id = $3
	`, request.Name, request.Color, columnID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update status"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Status not found"})
		return
	}
	orgID, _ := columnOrganization(c.Request.Context(), c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "column:updated", gin.H{"id": columnID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *ColumnHandler) Delete(c *gin.Context) {
	if !requireColumnRole(c, c.Param("id"), roleAdmin) {
		return
	}
	columnID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	var boardID int64
	var orgID int64
	if err := db.Pool.QueryRow(ctx, `SELECT c.board_id, b.org_id FROM columns c JOIN boards b ON b.id = c.board_id WHERE c.id = $1`, columnID).Scan(&boardID, &orgID); err != nil {
		respondMissing(c, err, "Status")
		return
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete status"})
		return
	}
	defer tx.Rollback(ctx)

	var taskCount int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE column_id = $1`, columnID).Scan(&taskCount); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete status"})
		return
	}
	if taskCount > 0 {
		c.JSON(409, models.ErrorResponse{Error: "Move all tasks out of this status before deleting it"})
		return
	}
	result, err := tx.Exec(ctx, `DELETE FROM columns WHERE id = $1`, columnID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete status"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Status not found"})
		return
	}
	if _, err = tx.Exec(ctx, `
		WITH ordered AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY position, id) - 1 AS new_position
			FROM columns WHERE board_id = $1
		)
		UPDATE columns c SET position = ordered.new_position
		FROM ordered WHERE c.id = ordered.id
	`, boardID); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete status"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete status"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "column:deleted", gin.H{"id": columnID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *ColumnHandler) Reorder(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleAdmin) {
		return
	}
	var request models.ColumnReorder
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid status order"})
		return
	}
	boardID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to reorder statuses"})
		return
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT id FROM columns WHERE board_id = $1 ORDER BY position, id FOR UPDATE`, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to reorder statuses"})
		return
	}
	existing := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			existing[id] = true
		}
	}
	rows.Close()
	if len(request.ColumnIDs) != len(existing) {
		c.JSON(400, models.ErrorResponse{Error: "Status order must contain every status exactly once"})
		return
	}
	seen := make(map[int64]bool, len(request.ColumnIDs))
	for _, id := range request.ColumnIDs {
		if !existing[id] || seen[id] {
			c.JSON(400, models.ErrorResponse{Error: "Status order must contain every status exactly once"})
			return
		}
		seen[id] = true
	}
	for position, id := range request.ColumnIDs {
		if _, err = tx.Exec(ctx, `UPDATE columns SET position = $1 WHERE id = $2 AND board_id = $3`, position, id, boardID); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to reorder statuses"})
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to reorder statuses"})
		return
	}
	orgID, _ := boardOrganization(c.Request.Context(), c.Param("id"), false)
	ws.BroadcastToOrgDirect(orgID, "columns:reordered", gin.H{"column_ids": request.ColumnIDs})
	c.JSON(200, gin.H{"ok": true})
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	_, err := hex.DecodeString(value[1:])
	return err == nil
}
