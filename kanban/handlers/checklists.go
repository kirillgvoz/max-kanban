package handlers

import (
	"strings"

	"kanban/db"
	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

type ChecklistHandler struct{}

func NewChecklistHandler() *ChecklistHandler { return &ChecklistHandler{} }

func (h *ChecklistHandler) Create(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.ChecklistCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid checklist data"})
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" {
		request.Title = "Чеклист"
	}
	taskID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist"})
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, taskID); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist"})
		return
	}
	var position int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM checklists WHERE task_id = $1`, taskID).Scan(&position); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist"})
		return
	}
	var checklist models.Checklist
	err = tx.QueryRow(ctx, `INSERT INTO checklists (task_id, title, position) VALUES ($1, $2, $3) RETURNING id, task_id, title, position`, taskID, request.Title, position).Scan(&checklist.ID, &checklist.TaskID, &checklist.Title, &checklist.Position)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist"})
		return
	}
	checklist.Items = make([]models.ChecklistItem, 0)
	orgID, _ := taskOrganization(ctx, c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "checklist:created", checklist)
	c.JSON(201, checklist)
}

func (h *ChecklistHandler) Update(c *gin.Context) {
	if !requireChecklistRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.ChecklistUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid checklist data"})
		return
	}
	if request.Title != nil {
		trimmed := strings.TrimSpace(*request.Title)
		request.Title = &trimmed
	}
	checklistID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `UPDATE checklists SET title = COALESCE($1, title) WHERE id = $2`, request.Title, checklistID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update checklist"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Checklist not found"})
		return
	}
	orgID, _ := checklistOrganization(c.Request.Context(), c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "checklist:updated", gin.H{"id": checklistID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *ChecklistHandler) Delete(c *gin.Context) {
	if !requireChecklistRole(c, c.Param("id"), roleMember) {
		return
	}
	checklistID, _ := positiveID(c.Param("id"))
	orgID, err := checklistOrganization(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondMissing(c, err, "Checklist")
		return
	}
	result, err := db.Pool.Exec(c.Request.Context(), `DELETE FROM checklists WHERE id = $1`, checklistID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete checklist"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Checklist not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "checklist:deleted", gin.H{"id": checklistID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *ChecklistHandler) CreateItem(c *gin.Context) {
	if !requireChecklistRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.ChecklistItemCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Checklist item text is required"})
		return
	}
	request.Text = strings.TrimSpace(request.Text)
	if request.Text == "" {
		c.JSON(400, models.ErrorResponse{Error: "Checklist item text is required"})
		return
	}
	checklistID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist item"})
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, checklistID); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist item"})
		return
	}
	var position int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM checklist_items WHERE checklist_id = $1`, checklistID).Scan(&position); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist item"})
		return
	}
	var item models.ChecklistItem
	err = tx.QueryRow(ctx, `INSERT INTO checklist_items (checklist_id, text, position) VALUES ($1, $2, $3) RETURNING id, checklist_id, text, is_done, position`, checklistID, request.Text, position).Scan(&item.ID, &item.ChecklistID, &item.Text, &item.IsDone, &item.Position)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist item"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create checklist item"})
		return
	}
	orgID, _ := checklistOrganization(ctx, c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "checklist:item:created", item)
	c.JSON(201, item)
}

func (h *ChecklistHandler) UpdateItem(c *gin.Context) {
	if !requireChecklistItemRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.ChecklistItemUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid checklist item"})
		return
	}
	if request.Text != nil {
		trimmed := strings.TrimSpace(*request.Text)
		request.Text = &trimmed
	}
	itemID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `UPDATE checklist_items SET text = COALESCE($1, text), is_done = COALESCE($2, is_done) WHERE id = $3`, request.Text, request.IsDone, itemID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update checklist item"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Checklist item not found"})
		return
	}
	orgID, _ := checklistItemOrganization(c.Request.Context(), c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "checklist:item:updated", gin.H{"id": itemID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *ChecklistHandler) DeleteItem(c *gin.Context) {
	if !requireChecklistItemRole(c, c.Param("id"), roleMember) {
		return
	}
	itemID, _ := positiveID(c.Param("id"))
	orgID, err := checklistItemOrganization(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondMissing(c, err, "Checklist item")
		return
	}
	result, err := db.Pool.Exec(c.Request.Context(), `DELETE FROM checklist_items WHERE id = $1`, itemID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete checklist item"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Checklist item not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "checklist:item:deleted", gin.H{"id": itemID})
	c.JSON(200, gin.H{"ok": true})
}
