package handlers

import (
	"context"
	"errors"
	"strconv"

	"kanban/db"
	"kanban/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const (
	roleMember = 1
	roleAdmin  = 2
	roleOwner  = 3
)

func requireOrgRole(c *gin.Context, orgID string, minimum int) bool {
	userID := getUserID(c)
	role, err := organizationRole(c.Request.Context(), userID, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(404, models.ErrorResponse{Error: "Organization not found"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to verify access"})
		}
		return false
	}
	if roleValue(role) < minimum {
		c.JSON(403, models.ErrorResponse{Error: "Insufficient permissions"})
		return false
	}
	return true
}

func requireBoardRole(c *gin.Context, boardID string, minimum int) bool {
	return requireBoardRoleWithArchived(c, boardID, minimum, false)
}

func requireBoardRoleWithArchived(c *gin.Context, boardID string, minimum int, includeArchived bool) bool {
	orgID, err := boardOrganization(c.Request.Context(), boardID, includeArchived)
	if err != nil {
		respondMissing(c, err, "Board")
		return false
	}
	return requireOrgRoleByID(c, orgID, minimum)
}

func requireColumnRole(c *gin.Context, columnID string, minimum int) bool {
	orgID, err := columnOrganization(c.Request.Context(), columnID)
	if err != nil {
		respondMissing(c, err, "Column")
		return false
	}
	return requireOrgRoleByID(c, orgID, minimum)
}

func requireTaskRole(c *gin.Context, taskID string, minimum int) bool {
	orgID, err := taskOrganization(c.Request.Context(), taskID)
	if err != nil {
		respondMissing(c, err, "Task")
		return false
	}
	return requireOrgRoleByID(c, orgID, minimum)
}

func requireChecklistRole(c *gin.Context, checklistID string, minimum int) bool {
	orgID, err := checklistOrganization(c.Request.Context(), checklistID)
	if err != nil {
		respondMissing(c, err, "Checklist")
		return false
	}
	return requireOrgRoleByID(c, orgID, minimum)
}

func requireChecklistItemRole(c *gin.Context, itemID string, minimum int) bool {
	orgID, err := checklistItemOrganization(c.Request.Context(), itemID)
	if err != nil {
		respondMissing(c, err, "Checklist item")
		return false
	}
	return requireOrgRoleByID(c, orgID, minimum)
}

func requireCommentAccess(c *gin.Context, commentID string) (int64, bool) {
	var taskID int64
	var authorID int64
	err := db.Pool.QueryRow(c.Request.Context(), `SELECT task_id, user_id FROM comments WHERE id = $1`, commentID).Scan(&taskID, &authorID)
	if err != nil {
		respondMissing(c, err, "Comment")
		return 0, false
	}
	if authorID == getUserID(c) {
		return taskID, true
	}
	orgID, err := taskOrganization(c.Request.Context(), strconv.FormatInt(taskID, 10))
	if err != nil {
		respondMissing(c, err, "Task")
		return 0, false
	}
	if !requireOrgRoleByID(c, orgID, roleAdmin) {
		return 0, false
	}
	return taskID, true
}

func requireOrgRoleByID(c *gin.Context, orgID int64, minimum int) bool {
	role, err := organizationRole(c.Request.Context(), getUserID(c), strconv.FormatInt(orgID, 10))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(404, models.ErrorResponse{Error: "Organization not found"})
		} else {
			c.JSON(500, models.ErrorResponse{Error: "Unable to verify access"})
		}
		return false
	}
	if roleValue(role) < minimum {
		c.JSON(403, models.ErrorResponse{Error: "Insufficient permissions"})
		return false
	}
	return true
}

func organizationRole(ctx context.Context, userID int64, orgID string) (string, error) {
	parsed, err := strconv.ParseInt(orgID, 10, 64)
	if err != nil || parsed <= 0 {
		return "", pgx.ErrNoRows
	}
	var role string
	err = db.Pool.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2`, parsed, userID).Scan(&role)
	return role, err
}

func boardOrganization(ctx context.Context, boardID string, includeArchived bool) (int64, error) {
	parsed, err := positiveID(boardID)
	if err != nil {
		return 0, err
	}
	query := `SELECT org_id FROM boards WHERE id = $1`
	if !includeArchived {
		query += ` AND is_archived = FALSE`
	}
	var orgID int64
	err = db.Pool.QueryRow(ctx, query, parsed).Scan(&orgID)
	return orgID, err
}

func columnOrganization(ctx context.Context, columnID string) (int64, error) {
	parsed, err := positiveID(columnID)
	if err != nil {
		return 0, err
	}
	var orgID int64
	err = db.Pool.QueryRow(ctx, `
		SELECT b.org_id FROM columns c
		JOIN boards b ON b.id = c.board_id
		WHERE c.id = $1 AND b.is_archived = FALSE
	`, parsed).Scan(&orgID)
	return orgID, err
}

func taskOrganization(ctx context.Context, taskID string) (int64, error) {
	parsed, err := positiveID(taskID)
	if err != nil {
		return 0, err
	}
	var orgID int64
	err = db.Pool.QueryRow(ctx, `
		SELECT b.org_id FROM tasks t
		JOIN boards b ON b.id = t.board_id
		WHERE t.id = $1 AND b.is_archived = FALSE
	`, parsed).Scan(&orgID)
	return orgID, err
}

func checklistOrganization(ctx context.Context, checklistID string) (int64, error) {
	parsed, err := positiveID(checklistID)
	if err != nil {
		return 0, err
	}
	var orgID int64
	err = db.Pool.QueryRow(ctx, `
		SELECT b.org_id FROM checklists cl
		JOIN tasks t ON t.id = cl.task_id
		JOIN boards b ON b.id = t.board_id
		WHERE cl.id = $1 AND b.is_archived = FALSE
	`, parsed).Scan(&orgID)
	return orgID, err
}

func checklistItemOrganization(ctx context.Context, itemID string) (int64, error) {
	parsed, err := positiveID(itemID)
	if err != nil {
		return 0, err
	}
	var orgID int64
	err = db.Pool.QueryRow(ctx, `
		SELECT b.org_id FROM checklist_items ci
		JOIN checklists cl ON cl.id = ci.checklist_id
		JOIN tasks t ON t.id = cl.task_id
		JOIN boards b ON b.id = t.board_id
		WHERE ci.id = $1 AND b.is_archived = FALSE
	`, parsed).Scan(&orgID)
	return orgID, err
}

func respondMissing(c *gin.Context, err error, resource string) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(404, models.ErrorResponse{Error: resource + " not found"})
		return
	}
	c.JSON(500, models.ErrorResponse{Error: "Database error"})
}

func positiveID(raw string) (int64, error) {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, pgx.ErrNoRows
	}
	return parsed, nil
}

func roleValue(role string) int {
	switch role {
	case "owner":
		return roleOwner
	case "admin":
		return roleAdmin
	default:
		return roleMember
	}
}
