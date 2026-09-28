package handlers

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"kanban/db"
	"kanban/services"
	"kanban/ws"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type WebhookHandler struct {
	Bot           *services.MaxBot
	WebhookSecret string
}

func NewWebhookHandler(bot *services.MaxBot, secret string) *WebhookHandler {
	return &WebhookHandler{Bot: bot, WebhookSecret: secret}
}

func (h *WebhookHandler) Handle(c *gin.Context) {
	if !hmac.Equal([]byte(c.GetHeader("X-Max-Bot-Api-Secret")), []byte(h.WebhookSecret)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	decoder.UseNumber()
	var update MaxUpdate
	if err := decoder.Decode(&update); err != nil {
		log.Printf("webhook decode error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	ctx := c.Request.Context()
	var err error
	switch update.UpdateType {
	case "message_created":
		err = h.handleMessage(ctx, update)
	case "message_callback":
		err = h.handleCallback(ctx, update)
	case "bot_started", "bot_added":
		err = h.handleBotStarted(ctx, update)
	default:
		log.Printf("unknown update type: %s", update.UpdateType)
	}
	if err != nil {
		log.Printf("webhook %s error: %v", update.UpdateType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *WebhookHandler) handleMessage(ctx context.Context, update MaxUpdate) error {
	text := maxText(update.Message)
	userID := update.Message.Sender.UserID.Int64()
	chatID, ok := maxMessageChat(update.Message)
	if update.Message == nil || text == "" || userID <= 0 || !ok {
		return fmt.Errorf("invalid message update")
	}
	raw, _ := json.Marshal(update)
	eventKey := webhookEventKey(update, raw)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	inserted, err := recordWebhookEvent(ctx, tx, eventKey, update, &chatID, &userID, nil, raw)
	if err != nil {
		return err
	}
	if !inserted {
		return tx.Commit(ctx)
	}

	switch {
	case strings.HasPrefix(text, "/link board_"):
		response, err := h.linkBoard(ctx, tx, text, userID, chatID)
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return h.sendChatMessage(ctx, chatID, response)
	case strings.HasPrefix(text, "/unlink board_"):
		response, err := h.unlinkBoard(ctx, tx, text, userID, chatID)
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return h.sendChatMessage(ctx, chatID, response)
	case strings.HasPrefix(text, "/new"):
		taskID, title, err := h.createTaskFromChat(ctx, tx, text, userID, chatID)
		if err != nil {
			return err
		}
		if taskID == 0 {
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return h.sendChatMessage(ctx, chatID, title)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		task, err := loadTask(ctx, taskID)
		if err != nil {
			return err
		}
		orgID, err := boardOrganization(ctx, strconv.FormatInt(task.BoardID, 10), false)
		if err != nil {
			return err
		}
		ws.BroadcastToOrgDirect(orgID, "task:created", task)
		if err := services.NotifyTaskCreated(context.Background(), task.BoardID, task.ID, task.Title); err != nil {
			log.Printf("queue chat task notification: %v", err)
		}
		return h.sendChatMessage(ctx, chatID, fmt.Sprintf("✅ Задача #%d создана: %s", taskID, title))
	case text == "/start", text == "/tasks":
		response, err := h.chatStatus(ctx, tx, text, chatID)
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return h.sendChatMessage(ctx, chatID, response)
	default:
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return h.sendChatMessage(ctx, chatID, "Используйте /link board_<id>, /tasks или /new <задача>")
	}
}

func (h *WebhookHandler) handleCallback(ctx context.Context, update MaxUpdate) error {
	if update.Callback == nil {
		return fmt.Errorf("missing callback")
	}
	action, taskID, ok := maxCallbackAction(strings.TrimSpace(update.Callback.Payload))
	if !ok {
		return fmt.Errorf("unsupported callback payload")
	}
	userID := update.Callback.User.UserID.Int64()
	chatID, hasChat := maxCallbackChat(update.Callback)
	if userID <= 0 || !hasChat || update.Callback.CallbackID == "" {
		return fmt.Errorf("invalid callback update")
	}
	raw, _ := json.Marshal(update)
	eventKey := webhookEventKey(update, raw)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	inserted, err := recordWebhookEvent(ctx, tx, eventKey, update, &chatID, &userID, &update.Callback.CallbackID, raw)
	if err != nil {
		return err
	}
	if !inserted {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return h.Bot.AnswerCallback(ctx, update.Callback.CallbackID, &services.OutgoingMessage{Text: "Действие уже обработано"})
	}

	var boardID, sourceColumnID, sourcePosition int64
	var title string
	if err := tx.QueryRow(ctx, `SELECT board_id, column_id, position, title FROM tasks WHERE id = $1 FOR UPDATE`, taskID).Scan(&boardID, &sourceColumnID, &sourcePosition, &title); err != nil {
		if err == pgx.ErrNoRows {
			return h.answerAndFail(ctx, tx, update.Callback.CallbackID, chatID, "Задача не найдена")
		}
		return err
	}
	var orgID int64
	if err := tx.QueryRow(ctx, `SELECT org_id FROM boards WHERE id = $1`, boardID).Scan(&orgID); err != nil {
		return err
	}
	var member bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND user_id = $2)`, orgID, userID).Scan(&member); err != nil {
		return err
	}
	if !member {
		return h.answerAndFail(ctx, tx, update.Callback.CallbackID, chatID, "Нет доступа к этой задаче")
	}

	var targetColumnID int64
	var targetPosition int
	var targetName string
	if action == "take" {
		if err := tx.QueryRow(ctx, `SELECT id, name FROM columns WHERE board_id = $1 AND position > $2 ORDER BY position LIMIT 1`, boardID, sourcePosition).Scan(&targetColumnID, &targetName); err != nil {
			if err == pgx.ErrNoRows {
				return h.answerAndFail(ctx, tx, update.Callback.CallbackID, chatID, "Задача уже в последнем статусе")
			}
			return err
		}
		targetPosition = 0
		movedIDs, err := orderedTaskIDs(ctx, tx, targetColumnID, 0)
		if err != nil {
			return err
		}
		targetPosition = len(movedIDs)
	} else {
		if err := tx.QueryRow(ctx, `SELECT id, name FROM columns WHERE board_id = $1 ORDER BY position DESC LIMIT 1`, boardID).Scan(&targetColumnID, &targetName); err != nil {
			return h.answerAndFail(ctx, tx, update.Callback.CallbackID, chatID, "Не найден финальный статус")
		}
		movedIDs, err := orderedTaskIDs(ctx, tx, targetColumnID, taskID)
		if err != nil {
			return err
		}
		targetPosition = len(movedIDs)
	}
	if err := moveTaskTx(ctx, tx, taskID, targetColumnID, targetPosition); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	updated, err := loadTask(ctx, taskID)
	if err != nil {
		return err
	}
	updated.Assignees, _ = loadAssignees(ctx, taskID)
	ws.BroadcastToOrgDirect(orgID, "task:moved", updated)
	if err := services.NotifyTaskStatus(ctx, boardID, taskID, targetColumnID, title, targetName); err != nil {
		log.Printf("queue callback status notification: %v", err)
	}
	return h.Bot.AnswerCallback(ctx, update.Callback.CallbackID, &services.OutgoingMessage{
		Text: fmt.Sprintf("Задача «%s» → %s", title, targetName),
	})
}

func (h *WebhookHandler) handleBotStarted(ctx context.Context, update MaxUpdate) error {
	chatID, ok := optionalMaxID(update.ChatID)
	userID := update.User.UserID.Int64()
	if !ok {
		return fmt.Errorf("invalid bot start update")
	}
	raw, _ := json.Marshal(update)
	eventKey := webhookEventKey(update, raw)
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	inserted, err := recordWebhookEvent(ctx, tx, eventKey, update, &chatID, &userID, nil, raw)
	if err != nil {
		return err
	}
	if !inserted {
		return tx.Commit(ctx)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return h.Bot.SendWelcome(ctx, chatID)
}

func (h *WebhookHandler) linkBoard(ctx context.Context, tx pgx.Tx, text string, userID, chatID int64) (string, error) {
	boardID, err := positiveID(strings.TrimPrefix(text, "/link board_"))
	if err != nil {
		return "Формат: /link board_<id>", nil
	}
	var orgID int64
	if err := tx.QueryRow(ctx, `SELECT org_id FROM boards WHERE id = $1 AND is_archived = FALSE`, boardID).Scan(&orgID); err != nil {
		if err == pgx.ErrNoRows {
			return "Доска не найдена", nil
		}
		return "", err
	}
	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return "Только участник организации может привязать доску", nil
		}
		return "", err
	}
	if role != "owner" && role != "admin" {
		return "Привязать доску может только owner или admin", nil
	}
	if _, err := tx.Exec(ctx, `INSERT INTO board_chats (board_id, chat_id, created_by) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, boardID, chatID, userID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Доска #%d привязана к этому чату", boardID), nil
}

func (h *WebhookHandler) unlinkBoard(ctx context.Context, tx pgx.Tx, text string, userID, chatID int64) (string, error) {
	boardID, err := positiveID(strings.TrimPrefix(text, "/unlink board_"))
	if err != nil {
		return "Формат: /unlink board_<id>", nil
	}
	var orgID int64
	if err := tx.QueryRow(ctx, `SELECT org_id FROM boards WHERE id = $1`, boardID).Scan(&orgID); err != nil {
		if err == pgx.ErrNoRows {
			return "Доска не найдена", nil
		}
		return "", err
	}
	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return "Только участник организации может отвязать доску", nil
		}
		return "", err
	}
	if role != "owner" && role != "admin" {
		return "Отвязать доску может только owner или admin", nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM board_chats WHERE board_id = $1 AND chat_id = $2`, boardID, chatID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Доска #%d отвязана от этого чата", boardID), nil
}

func (h *WebhookHandler) createTaskFromChat(ctx context.Context, tx pgx.Tx, text string, userID, chatID int64) (int64, string, error) {
	title := strings.TrimSpace(strings.TrimPrefix(text, "/new"))
	if title == "" {
		return 0, "Формат: /new Название задачи", nil
	}
	rows, err := tx.Query(ctx, `
		SELECT b.id, b.org_id FROM board_chats bc
		JOIN boards b ON b.id = bc.board_id
		WHERE bc.chat_id = $1 AND b.is_archived = FALSE
		ORDER BY b.id
	`, chatID)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	type linkedBoard struct {
		ID    int64
		OrgID int64
	}
	boards := make([]linkedBoard, 0)
	for rows.Next() {
		var board linkedBoard
		if err := rows.Scan(&board.ID, &board.OrgID); err != nil {
			return 0, "", err
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	if len(boards) == 0 {
		return 0, "Сначала привяжите доску: /link board_<id>", nil
	}
	if len(boards) > 1 {
		return 0, "К чату привязано несколько досок. Пока поддерживается одна доска на чат для /new", nil
	}
	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2`, boards[0].OrgID, userID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return 0, "Создавать задачи может только участник организации", nil
		}
		return 0, "", err
	}
	var columnID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM columns WHERE board_id = $1 ORDER BY position, id LIMIT 1`, boards[0].ID).Scan(&columnID); err != nil {
		if err == pgx.ErrNoRows {
			return 0, "На доске нет статусов", nil
		}
		return 0, "", err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, columnID); err != nil {
		return 0, "", err
	}
	var position int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM tasks WHERE column_id = $1`, columnID).Scan(&position); err != nil {
		return 0, "", err
	}
	var taskID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO tasks (board_id, column_id, title, description, position, priority, created_by)
		VALUES ($1, $2, $3, '', $4, 'medium', $5)
		RETURNING id
	`, boards[0].ID, columnID, title, position, userID).Scan(&taskID); err != nil {
		return 0, "", err
	}
	return taskID, title, nil
}

func (h *WebhookHandler) chatStatus(ctx context.Context, tx pgx.Tx, text string, chatID int64) (string, error) {
	if text == "/start" {
		return "👋 Добро пожаловать в TaskFlow! Используйте /link board_<id> в рабочем чате, чтобы подключить доску.", nil
	}
	rows, err := tx.Query(ctx, `
		SELECT b.id, b.name, COUNT(t.id) FILTER (WHERE t.deadline IS NOT NULL AND t.deadline <= CURRENT_DATE + 7)
		FROM board_chats bc
		JOIN boards b ON b.id = bc.board_id
		LEFT JOIN tasks t ON t.board_id = b.id
		WHERE bc.chat_id = $1 AND b.is_archived = FALSE
		GROUP BY b.id, b.name
		ORDER BY b.name
	`, chatID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var id, due int64
		var name string
		if err := rows.Scan(&id, &name, &due); err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf("• #%d %s — дедлайны на 7 дней: %d", id, name, due))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "К чату пока не привязана ни одна доска. Используйте /link board_<id>", nil
	}
	return "Доски этого чата:\n" + strings.Join(lines, "\n"), nil
}

func (h *WebhookHandler) sendChatMessage(ctx context.Context, chatID int64, text string) error {
	return h.Bot.SendChatMessage(ctx, chatID, services.OutgoingMessage{
		Text: text,
		Attachments: []any{services.InlineKeyboard([][]services.Button{
			{services.OpenAppButton("📋 Открыть TaskFlow", h.Bot.FrontendURL)},
		})},
	})
}

func (h *WebhookHandler) answerAndFail(ctx context.Context, tx pgx.Tx, callbackID string, chatID int64, text string) error {
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if err := h.Bot.AnswerCallback(ctx, callbackID, &services.OutgoingMessage{Text: text}); err != nil {
		return err
	}
	return h.Bot.SendChatMessage(ctx, chatID, services.OutgoingMessage{Text: text})
}

func recordWebhookEvent(ctx context.Context, tx pgx.Tx, key string, update MaxUpdate, chatID, userID *int64, callbackID *string, raw []byte) (bool, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO webhook_events (event_key, update_type, chat_id, user_id, callback_id, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_key) DO NOTHING
		RETURNING id
	`, key, update.UpdateType, chatID, userID, callbackID, raw).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
