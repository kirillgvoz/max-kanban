package handlers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"kanban/db"
	"kanban/models"
	"kanban/services"
	"kanban/ws"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type TaskHandler struct{}

func NewTaskHandler() *TaskHandler { return &TaskHandler{} }

type rowScanner interface {
	Scan(dest ...any) error
}

func (h *TaskHandler) ListByBoard(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleMember) {
		return
	}
	boardID, _ := positiveID(c.Param("id"))
	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT id, board_id, column_id, title, description, position, priority,
		       COALESCE(deadline::text, ''), created_by, created_at, updated_at
		FROM tasks WHERE board_id = $1
		ORDER BY column_id, position, id
	`, boardID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load tasks"})
		return
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load tasks"})
			return
		}
		assignees, err := loadAssignees(c.Request.Context(), task.ID)
		if err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load tasks"})
			return
		}
		task.Assignees = assignees
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load tasks"})
		return
	}
	c.JSON(200, tasks)
}

func (h *TaskHandler) Get(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	var detail models.TaskDetail
	var err error
	detail.Task, err = loadTask(ctx, taskID)
	if err != nil {
		respondMissing(c, err, "Task")
		return
	}
	detail.Assignees, _ = loadAssignees(ctx, detail.ID)
	detail.Checklists = make([]models.Checklist, 0)
	detail.Comments = make([]models.Comment, 0)

	checklistRows, err := db.Pool.Query(ctx, `SELECT id, task_id, title, position FROM checklists WHERE task_id = $1 ORDER BY position, id`, detail.ID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
		return
	}
	for checklistRows.Next() {
		var checklist models.Checklist
		if err := checklistRows.Scan(&checklist.ID, &checklist.TaskID, &checklist.Title, &checklist.Position); err != nil {
			checklistRows.Close()
			c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
			return
		}
		checklist.Items = make([]models.ChecklistItem, 0)
		itemRows, itemErr := db.Pool.Query(ctx, `SELECT id, checklist_id, text, is_done, position FROM checklist_items WHERE checklist_id = $1 ORDER BY position, id`, checklist.ID)
		if itemErr != nil {
			checklistRows.Close()
			c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
			return
		}
		for itemRows.Next() {
			var item models.ChecklistItem
			if err := itemRows.Scan(&item.ID, &item.ChecklistID, &item.Text, &item.IsDone, &item.Position); err != nil {
				itemRows.Close()
				checklistRows.Close()
				c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
				return
			}
			checklist.Items = append(checklist.Items, item)
		}
		itemRows.Close()
		detail.Checklists = append(detail.Checklists, checklist)
	}
	checklistRows.Close()

	commentRows, err := db.Pool.Query(ctx, `SELECT id, task_id, user_id, username, display_name, text, created_at FROM comments WHERE task_id = $1 ORDER BY created_at, id`, detail.ID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
		return
	}
	defer commentRows.Close()
	for commentRows.Next() {
		var comment models.Comment
		if err := commentRows.Scan(&comment.ID, &comment.TaskID, &comment.UserID, &comment.Username, &comment.DisplayName, &comment.Text, &comment.CreatedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load task"})
			return
		}
		detail.Comments = append(detail.Comments, comment)
	}
	c.JSON(200, detail)
}

func (h *TaskHandler) Create(c *gin.Context) {
	if !requireBoardRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.TaskCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid task data"})
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" {
		c.JSON(400, models.ErrorResponse{Error: "Task title is required"})
		return
	}
	if request.Priority == "" {
		request.Priority = "medium"
	}
	deadline, err := normalizeDeadline(request.Deadline)
	if err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Deadline must use YYYY-MM-DD"})
		return
	}

	boardID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		return
	}
	defer tx.Rollback(ctx)

	columnID := request.ColumnID
	if columnID == 0 {
		if err = tx.QueryRow(ctx, `SELECT id FROM columns WHERE board_id = $1 ORDER BY position, id LIMIT 1`, boardID).Scan(&columnID); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(409, models.ErrorResponse{Error: "Create at least one status before adding tasks"})
			} else {
				c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
			}
			return
		}
	}
	var columnBoardID int64
	if err = tx.QueryRow(ctx, `SELECT board_id FROM columns WHERE id = $1`, columnID).Scan(&columnBoardID); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(400, models.ErrorResponse{Error: "Status does not exist"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		}
		return
	}
	if columnBoardID != boardID {
		c.JSON(400, models.ErrorResponse{Error: "Status belongs to another board"})
		return
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, columnID); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		return
	}
	var position int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM tasks WHERE column_id = $1`, columnID).Scan(&position); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		return
	}
	var task models.Task
	var deadlineText string
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (board_id, column_id, title, description, position, priority, deadline, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, board_id, column_id, title, description, position, priority,
		          COALESCE(deadline::text, ''), created_by, created_at, updated_at
	`, boardID, columnID, request.Title, request.Description, position, request.Priority, deadline, getUserID(c)).Scan(
		&task.ID, &task.BoardID, &task.ColumnID, &task.Title, &task.Description, &task.Position,
		&task.Priority, &deadlineText, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
	)
	if deadlineText != "" {
		task.Deadline = &deadlineText
	}
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create task"})
		return
	}
	task.Assignees = make([]models.Assignee, 0)
	orgID, _ := boardOrganization(ctx, c.Param("id"), false)
	if err := services.NotifyTaskCreated(ctx, task.BoardID, task.ID, task.Title); err != nil {
		log.Printf("queue task notification: %v", err)
	}
	ws.BroadcastToOrgDirect(orgID, "task:created", task)
	c.JSON(201, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.TaskUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid task data"})
		return
	}
	if request.Title != nil {
		trimmed := strings.TrimSpace(*request.Title)
		request.Title = &trimmed
	}
	var deadline any
	if request.Deadline.Present && request.Deadline.Valid {
		normalized, err := normalizeDeadline(&request.Deadline.Value)
		if err != nil {
			c.JSON(400, models.ErrorResponse{Error: "Deadline must use YYYY-MM-DD"})
			return
		}
		deadline = normalized
	}
	taskID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `
		UPDATE tasks
		SET title = COALESCE($1, title),
		    description = COALESCE($2, description),
		    priority = COALESCE($3, priority),
		    deadline = CASE WHEN $4::boolean THEN $5::date ELSE deadline END,
		    updated_at = NOW()
		WHERE id = $6
	`, request.Title, request.Description, request.Priority, request.Deadline.Present, deadline, taskID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update task"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Task not found"})
		return
	}
	task, err := loadTask(c.Request.Context(), taskID)
	if err != nil {
		respondMissing(c, err, "Task")
		return
	}
	task.Assignees, _ = loadAssignees(c.Request.Context(), task.ID)
	orgID, _ := taskOrganization(c.Request.Context(), c.Param("id"))
	ws.BroadcastToOrgDirect(orgID, "task:updated", task)
	c.JSON(200, task)
}

func (h *TaskHandler) Move(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	var request models.TaskMove
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "column_id and position are required"})
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to move task"})
		return
	}
	defer tx.Rollback(ctx)

	var boardID int64
	var sourceColumnID int64
	err = tx.QueryRow(ctx, `SELECT board_id, column_id FROM tasks WHERE id = $1 FOR UPDATE`, taskID).Scan(&boardID, &sourceColumnID)
	if err != nil {
		respondMissing(c, err, "Task")
		return
	}
	var targetBoardID int64
	if err = tx.QueryRow(ctx, `SELECT board_id FROM columns WHERE id = $1`, request.ColumnID).Scan(&targetBoardID); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(400, models.ErrorResponse{Error: "Status does not exist"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to move task"})
		}
		return
	}
	if targetBoardID != boardID {
		c.JSON(400, models.ErrorResponse{Error: "Status belongs to another board"})
		return
	}

	targetPosition := *request.Position
	if err := moveTaskTx(ctx, tx, taskID, request.ColumnID, targetPosition); err != nil {
		if err == errInvalidMove {
			c.JSON(400, models.ErrorResponse{Error: "Position is outside the status"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to move task"})
		}
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to move task"})
		return
	}
	task, err := loadTask(ctx, taskID)
	if err != nil {
		respondMissing(c, err, "Task")
		return
	}
	task.Assignees, _ = loadAssignees(ctx, task.ID)
	var status string
	_ = db.Pool.QueryRow(ctx, `SELECT name FROM columns WHERE id = $1`, task.ColumnID).Scan(&status)
	orgID, _ := boardOrganization(ctx, c.Param("id"), false)
	if err := services.NotifyTaskStatus(ctx, task.BoardID, task.ID, task.ColumnID, task.Title, status, sourceColumnID, getUserID(c)); err != nil {
		log.Printf("queue status notification: %v", err)
	}
	ws.BroadcastToOrgDirect(orgID, "task:moved", task)
	c.JSON(200, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	orgID, err := taskOrganization(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondMissing(c, err, "Task")
		return
	}
	result, err := db.Pool.Exec(c.Request.Context(), `DELETE FROM tasks WHERE id = $1`, taskID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to delete task"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Task not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "task:deleted", gin.H{"id": taskID})
	c.JSON(200, gin.H{"ok": true})
}

func (h *TaskHandler) Assign(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	var request struct {
		UserID int64 `json:"user_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "user_id is required"})
		return
	}
	ctx := c.Request.Context()
	var orgID int64
	var title string
	if err := db.Pool.QueryRow(ctx, `
		SELECT b.org_id, t.title FROM tasks t JOIN boards b ON b.id = t.board_id WHERE t.id = $1
	`, c.Param("id")).Scan(&orgID, &title); err != nil {
		respondMissing(c, err, "Task")
		return
	}
	var username, displayName string
	if err := db.Pool.QueryRow(ctx, `SELECT username, display_name FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, request.UserID).Scan(&username, &displayName); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(400, models.ErrorResponse{Error: "Assignee must be an organization member"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to assign task"})
		}
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO task_assignees (task_id, user_id, username, display_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, user_id) DO UPDATE SET username = EXCLUDED.username, display_name = EXCLUDED.display_name
	`, taskID, request.UserID, username, displayName); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to assign task"})
		return
	}
	assignee := models.Assignee{TaskID: taskID, UserID: request.UserID, Username: username, DisplayName: displayName}
	if err := services.NotifyTaskAssigned(ctx, taskID, request.UserID, title); err != nil {
		log.Printf("queue assignment notification: %v", err)
	}
	ws.BroadcastToOrgDirect(orgID, "task:assigned", gin.H{"task_id": taskID, "assignee": assignee})
	c.JSON(200, assignee)
}

func (h *TaskHandler) Unassign(c *gin.Context) {
	if !requireTaskRole(c, c.Param("id"), roleMember) {
		return
	}
	taskID, _ := positiveID(c.Param("id"))
	userID, _ := positiveID(c.Param("uid"))
	orgID, _ := taskOrganization(c.Request.Context(), c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `DELETE FROM task_assignees WHERE task_id = $1 AND user_id = $2`, taskID, userID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to remove assignee"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Assignee not found"})
		return
	}
	ws.BroadcastToOrgDirect(orgID, "task:unassigned", gin.H{"task_id": taskID, "user_id": userID})
	c.JSON(200, gin.H{"ok": true})
}

func scanTask(row rowScanner) (models.Task, error) {
	var task models.Task
	var deadline string
	err := row.Scan(
		&task.ID, &task.BoardID, &task.ColumnID, &task.Title, &task.Description,
		&task.Position, &task.Priority, &deadline, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
	)
	if deadline != "" {
		task.Deadline = &deadline
	}
	task.Assignees = make([]models.Assignee, 0)
	return task, err
}

func loadTask(ctx context.Context, taskID int64) (models.Task, error) {
	return scanTask(db.Pool.QueryRow(ctx, `
		SELECT id, board_id, column_id, title, description, position, priority,
		       COALESCE(deadline::text, ''), created_by, created_at, updated_at
		FROM tasks WHERE id = $1
	`, taskID))
}

func loadAssignees(ctx context.Context, taskID int64) ([]models.Assignee, error) {
	rows, err := db.Pool.Query(ctx, `SELECT task_id, user_id, username, display_name FROM task_assignees WHERE task_id = $1 ORDER BY user_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignees := make([]models.Assignee, 0)
	for rows.Next() {
		var assignee models.Assignee
		if err := rows.Scan(&assignee.TaskID, &assignee.UserID, &assignee.Username, &assignee.DisplayName); err != nil {
			return nil, err
		}
		assignees = append(assignees, assignee)
	}
	return assignees, rows.Err()
}

var errInvalidMove = errors.New("invalid move")

func moveTaskTx(ctx context.Context, tx pgx.Tx, taskID, targetColumnID int64, targetPosition int) error {
	var sourceColumnID int64
	if err := tx.QueryRow(ctx, `SELECT column_id FROM tasks WHERE id = $1 FOR UPDATE`, taskID).Scan(&sourceColumnID); err != nil {
		return err
	}
	if sourceColumnID == targetColumnID {
		ids, err := orderedTaskIDs(ctx, tx, targetColumnID, taskID)
		if err != nil {
			return err
		}
		if targetPosition < 0 || (len(ids) > 0 && targetPosition >= len(ids)) {
			return errInvalidMove
		}
		ordered := make([]int64, 0, len(ids)+1)
		ordered = append(ordered, ids[:targetPosition]...)
		ordered = append(ordered, taskID)
		ordered = append(ordered, ids[targetPosition:]...)
		for position, id := range ordered {
			if _, err := tx.Exec(ctx, `UPDATE tasks SET column_id = $1, position = $2, updated_at = NOW() WHERE id = $3`, targetColumnID, position, id); err != nil {
				return err
			}
		}
		return nil
	}
	ids, err := orderedTaskIDs(ctx, tx, targetColumnID, 0)
	if err != nil {
		return err
	}
	if targetPosition < 0 || targetPosition > len(ids) {
		return errInvalidMove
	}
	if _, err := tx.Exec(ctx, `UPDATE tasks SET position = position + 1 WHERE column_id = $1 AND position >= $2`, targetColumnID, targetPosition); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tasks SET column_id = $1, position = $2, updated_at = NOW() WHERE id = $3`, targetColumnID, targetPosition, taskID); err != nil {
		return err
	}
	return normalizeTaskPositions(ctx, tx, sourceColumnID)
}

func orderedTaskIDs(ctx context.Context, tx pgx.Tx, columnID int64, excludeTaskID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `SELECT id FROM tasks WHERE column_id = $1 AND id <> $2 ORDER BY position, id FOR UPDATE`, columnID, excludeTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func normalizeTaskPositions(ctx context.Context, tx pgx.Tx, columnID int64) error {
	ids, err := orderedTaskIDs(ctx, tx, columnID, 0)
	if err != nil {
		return err
	}
	for position, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE tasks SET position = $1 WHERE id = $2`, position, id); err != nil {
			return err
		}
	}
	return nil
}

func normalizeDeadline(value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	normalized := strings.TrimSpace(*value)
	if _, err := time.Parse("2006-01-02", normalized); err != nil {
		return nil, err
	}
	return &normalized, nil
}
