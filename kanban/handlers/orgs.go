package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"kanban/db"
	"kanban/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type OrgHandler struct{}

func NewOrgHandler() *OrgHandler { return &OrgHandler{} }

func (h *OrgHandler) List(c *gin.Context) {
	rows, err := db.Pool.Query(c.Request.Context(), `
		SELECT o.id, o.name, o.slug, COALESCE(o.avatar_url, ''), o.created_by, o.created_at
		FROM organizations o
		JOIN org_members m ON m.org_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.name
	`, getUserID(c))
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load organizations"})
		return
	}
	defer rows.Close()

	organizations := make([]models.Organization, 0)
	for rows.Next() {
		var organization models.Organization
		if err := rows.Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.AvatarURL, &organization.CreatedBy, &organization.CreatedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load organizations"})
			return
		}
		organizations = append(organizations, organization)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load organizations"})
		return
	}
	c.JSON(200, organizations)
}

func (h *OrgHandler) Create(c *gin.Context) {
	var request models.OrgCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Organization name must contain 2-80 characters"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if len(request.Name) < 2 {
		c.JSON(400, models.ErrorResponse{Error: "Organization name must contain 2-80 characters"})
		return
	}

	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}
	defer tx.Rollback(ctx)

	baseSlug := generateSlug(request.Name)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, baseSlug); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}
	slug, err := availableSlug(ctx, tx, baseSlug)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}

	userID := getUserID(c)
	var organization models.Organization
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, created_by, created_at
	`, request.Name, slug, userID).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.CreatedBy, &organization.CreatedAt)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO org_members (org_id, user_id, username, display_name, role)
		VALUES ($1, $2, $3, $4, 'owner')
	`, organization.ID, userID, getUsername(c), getDisplayName(c))
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to create organization"})
		return
	}
	c.JSON(201, organization)
}

func (h *OrgHandler) Get(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleMember) {
		return
	}
	orgID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()

	var organization models.Organization
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(avatar_url, ''), created_by, created_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.AvatarURL, &organization.CreatedBy, &organization.CreatedAt)
	if err != nil {
		respondMissing(c, err, "Organization")
		return
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT org_id, user_id, COALESCE(username, ''), COALESCE(display_name, ''), role, joined_at
		FROM org_members WHERE org_id = $1 ORDER BY joined_at
	`, orgID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load organization"})
		return
	}
	defer rows.Close()

	members := make([]models.OrgMember, 0)
	for rows.Next() {
		var member models.OrgMember
		if err := rows.Scan(&member.OrgID, &member.UserID, &member.Username, &member.DisplayName, &member.Role, &member.JoinedAt); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load organization"})
			return
		}
		members = append(members, member)
	}
	c.JSON(200, models.OrgDetail{Organization: organization, Members: members})
}

func (h *OrgHandler) Update(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleOwner) {
		return
	}
	var request models.OrgUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid organization data"})
		return
	}
	if request.Name != nil {
		trimmed := strings.TrimSpace(*request.Name)
		request.Name = &trimmed
	}
	orgID, _ := positiveID(c.Param("id"))
	result, err := db.Pool.Exec(c.Request.Context(), `
		UPDATE organizations
		SET name = COALESCE($1, name), avatar_url = COALESCE($2, avatar_url)
		WHERE id = $3
	`, request.Name, request.AvatarURL, orgID)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to update organization"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(404, models.ErrorResponse{Error: "Organization not found"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (h *OrgHandler) AddMember(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleOwner) {
		return
	}
	var request models.MemberCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, models.ErrorResponse{Error: "Invalid member data"})
		return
	}
	if request.Role == "" {
		request.Role = "member"
	}
	orgID, _ := positiveID(c.Param("id"))
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to add member"})
		return
	}
	defer tx.Rollback(ctx)

	var currentRole string
	err = tx.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2 FOR UPDATE`, orgID, request.UserID).Scan(&currentRole)
	if err != nil && err != pgx.ErrNoRows {
		c.JSON(500, models.ErrorResponse{Error: "Unable to add member"})
		return
	}
	if currentRole == "owner" && request.Role != "owner" {
		var owners int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'owner'`, orgID).Scan(&owners); err != nil || owners <= 1 {
			c.JSON(409, models.ErrorResponse{Error: "The organization must keep at least one owner"})
			return
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO org_members (org_id, user_id, username, display_name, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (org_id, user_id) DO UPDATE
		SET username = EXCLUDED.username, display_name = EXCLUDED.display_name, role = EXCLUDED.role
	`, orgID, request.UserID, request.Username, request.Name, request.Role)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to add member"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to add member"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (h *OrgHandler) RemoveMember(c *gin.Context) {
	if !requireOrgRole(c, c.Param("id"), roleOwner) {
		return
	}
	orgID, _ := positiveID(c.Param("id"))
	userID, err := positiveID(c.Param("uid"))
	if err != nil {
		c.JSON(404, models.ErrorResponse{Error: "Member not found"})
		return
	}
	ctx := c.Request.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to remove member"})
		return
	}
	defer tx.Rollback(ctx)

	var role string
	if err = tx.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2 FOR UPDATE`, orgID, userID).Scan(&role); err != nil {
		respondMissing(c, err, "Member")
		return
	}
	if role == "owner" {
		var owners int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'owner'`, orgID).Scan(&owners); err != nil || owners <= 1 {
			c.JSON(409, models.ErrorResponse{Error: "The organization must keep at least one owner"})
			return
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to remove member"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to remove member"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func getUserID(c *gin.Context) int64 {
	if value, exists := c.Get("user_id"); exists {
		if id, ok := value.(int64); ok {
			return id
		}
	}
	return 0
}

func getUsername(c *gin.Context) string {
	return c.GetString("username")
}

func getDisplayName(c *gin.Context) string {
	return c.GetString("display_name")
}

func availableSlug(ctx context.Context, tx pgx.Tx, base string) (string, error) {
	candidate := base
	for suffix := 2; ; suffix++ {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.NewReplacer(
		"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e", "ё", "e", "ж", "zh",
		"з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m", "н", "n", "о", "o",
		"п", "p", "р", "r", "с", "s", "т", "t", "у", "u", "ф", "f", "х", "h", "ц", "ts",
		"ч", "ch", "ш", "sh", "щ", "sch", "ъ", "", "ы", "y", "ь", "", "э", "e", "ю", "yu", "я", "ya",
	).Replace(slug)
	slug = strings.Join(strings.Fields(slug), "-")
	slug = slugRe.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "organization"
	}
	return slug
}
