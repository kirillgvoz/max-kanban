package handlers

import (
	"context"
	"time"

	"kanban/db"
	"kanban/models"
	"kanban/ws"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	Version   string
	Revision  string
	StartedAt time.Time
}

func NewHealthHandler(version, revision string, startedAt time.Time) *HealthHandler {
	return &HealthHandler{Version: version, Revision: revision, StartedAt: startedAt}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	database := "ok"
	status := 200
	if err := db.Pool.Ping(ctx); err != nil {
		database = "unavailable"
		status = 503
	}
	c.JSON(status, gin.H{
		"status":   map[bool]string{true: "ok", false: "degraded"}[status == 200],
		"version":  h.Version,
		"revision": h.Revision,
		"uptime":   time.Since(h.StartedAt).Truncate(time.Second).String(),
		"database": database,
	})
}

func (h *HealthHandler) Metrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	counts := map[string]int{"pending": 0, "sent": 0, "failed": 0}
	rows, err := db.Pool.Query(ctx, `SELECT status, COUNT(*) FROM notification_outbox GROUP BY status`)
	if err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load metrics"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			c.JSON(500, models.ErrorResponse{Error: "Unable to load metrics"})
			return
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, models.ErrorResponse{Error: "Unable to load metrics"})
		return
	}
	c.JSON(200, gin.H{
		"websocket_clients":   ws.DefaultHub.Total(),
		"notification_outbox": counts,
		"schema_migrations":   migrationCount(ctx),
	})
}

func migrationCount(ctx context.Context) int {
	var count int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return -1
	}
	return count
}
