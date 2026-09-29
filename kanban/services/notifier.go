package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationDestination struct {
	ChatID *int64
	UserID *int64
}

type NotificationSender interface {
	SendNotification(ctx context.Context, destination NotificationDestination, message OutgoingMessage) error
}

type Notifier struct {
	Pool       *pgxpool.Pool
	Sender     NotificationSender
	Now        func() time.Time
	Interval   time.Duration
	RateLimit  time.Duration
	MaxRetries int
}

type queuedNotification struct {
	ID          int64
	EventKey    string
	Destination NotificationDestination
	Message     OutgoingMessage
	Attempts    int
}

func NewNotifier(pool *pgxpool.Pool, sender NotificationSender) *Notifier {
	return &Notifier{
		Pool:       pool,
		Sender:     sender,
		Now:        time.Now,
		Interval:   5 * time.Second,
		RateLimit:  500 * time.Millisecond,
		MaxRetries: 5,
	}
}

var DefaultNotifier *Notifier

func SetDefaultNotifier(notifier *Notifier) {
	DefaultNotifier = notifier
}

func NotifyTaskCreated(ctx context.Context, boardID, taskID int64, title string) error {
	if DefaultNotifier == nil {
		return nil
	}
	return DefaultNotifier.EnqueueTaskCreated(ctx, boardID, taskID, title)
}

func NotifyTaskStatus(ctx context.Context, boardID, taskID, columnID int64, title, status string, prevColumnID, actorID int64) error {
	if DefaultNotifier == nil {
		return nil
	}
	return DefaultNotifier.EnqueueTaskStatus(ctx, boardID, taskID, columnID, title, status, prevColumnID, actorID)
}

func NotifyTaskAssigned(ctx context.Context, taskID, userID int64, title string) error {
	if DefaultNotifier == nil {
		return nil
	}
	return DefaultNotifier.EnqueueTaskAssigned(ctx, taskID, userID, title)
}

func (n *Notifier) Enqueue(ctx context.Context, eventKey string, destination NotificationDestination, message OutgoingMessage) (bool, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return false, err
	}
	var chatID, userID *int64
	if destination.ChatID != nil {
		chatID = destination.ChatID
	}
	if destination.UserID != nil {
		userID = destination.UserID
	}
	result, err := n.Pool.Exec(ctx, `
		INSERT INTO notification_outbox (event_key, chat_id, user_id, text, payload)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_key) DO NOTHING
	`, eventKey, chatID, userID, message.Text, payload)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

func (n *Notifier) Run(ctx context.Context) error {
	ticker := time.NewTicker(n.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := n.ProcessOnce(ctx); err != nil {
				log.Printf("notification worker error: %v", err)
			}
		}
	}
}

func (n *Notifier) ProcessOnce(ctx context.Context) error {
	notification, err := n.claimNext(ctx)
	if err != nil {
		return err
	}
	if notification == nil {
		return nil
	}
	if err := n.Sender.SendNotification(ctx, notification.Destination, notification.Message); err != nil {
		return n.markFailed(ctx, notification, err)
	}
	_, err = n.Pool.Exec(ctx, `UPDATE notification_outbox SET status = 'sent', last_error = '' WHERE id = $1`, notification.ID)
	time.Sleep(n.RateLimit)
	return err
}

func (n *Notifier) claimNext(ctx context.Context) (*queuedNotification, error) {
	tx, err := n.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var notification queuedNotification
	var chatID, userID *int64
	var payload []byte
	err = tx.QueryRow(ctx, `
		UPDATE notification_outbox
		SET attempts = attempts + 1, next_attempt_at = NOW()
		WHERE id = (
			SELECT id FROM notification_outbox
			WHERE status = 'pending' AND next_attempt_at <= NOW()
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, event_key, chat_id, user_id, payload, attempts
	`).Scan(&notification.ID, &notification.EventKey, &chatID, &userID, &payload, &notification.Attempts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(payload, &notification.Message); err != nil {
		return nil, err
	}
	notification.Destination = NotificationDestination{ChatID: chatID, UserID: userID}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &notification, nil
}

func (n *Notifier) markFailed(ctx context.Context, notification *queuedNotification, sendErr error) error {
	next := n.Now().Add(time.Duration(notification.Attempts*notification.Attempts) * time.Minute)
	if next.Sub(n.Now()) > time.Hour {
		next = n.Now().Add(time.Hour)
	}
	status := "pending"
	if notification.Attempts >= n.MaxRetries {
		status = "failed"
	}
	_, err := n.Pool.Exec(ctx, `
		UPDATE notification_outbox
		SET status = $1, next_attempt_at = $2, last_error = $3
		WHERE id = $4
	`, status, next, sendErr.Error(), notification.ID)
	return err
}

// taskCard carries the display context of a task for notifications.
type taskCard struct {
	Board    string
	Status   string
	Deadline string // DD.MM or empty
}

func (n *Notifier) loadTaskCard(ctx context.Context, taskID int64) (taskCard, error) {
	var card taskCard
	var deadline *string
	err := n.Pool.QueryRow(ctx, `
		SELECT b.name,
			COALESCE((SELECT c.name FROM columns c WHERE c.id = t.column_id), ''),
			t.deadline::text
		FROM tasks t
		JOIN boards b ON b.id = t.board_id
		WHERE t.id = $1
	`, taskID).Scan(&card.Board, &card.Status, &deadline)
	if err != nil {
		return taskCard{}, err
	}
	if deadline != nil {
		card.Deadline = formatShortDate(*deadline)
	}
	return card, nil
}

func (n *Notifier) memberDisplayName(ctx context.Context, boardID, userID int64) string {
	if userID <= 0 {
		return ""
	}
	var orgID int64
	if err := n.Pool.QueryRow(ctx, `SELECT org_id FROM boards WHERE id = $1`, boardID).Scan(&orgID); err != nil {
		return ""
	}
	var name string
	if err := n.Pool.QueryRow(ctx, `SELECT display_name FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID).Scan(&name); err != nil {
		return ""
	}
	return name
}

func (n *Notifier) columnName(ctx context.Context, columnID int64) string {
	if columnID <= 0 {
		return ""
	}
	var name string
	if err := n.Pool.QueryRow(ctx, `SELECT name FROM columns WHERE id = $1`, columnID).Scan(&name); err != nil {
		return ""
	}
	return name
}

// joinContext joins non-empty context parts with a middle dot.
func joinContext(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, " · ")
}

// formatShortDate renders an ISO date (YYYY-MM-DD) as DD.MM.
func formatShortDate(iso string) string {
	parsed, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return parsed.Format("02.01")
}

func formatTaskCreated(taskID int64, title string, card taskCard) string {
	head := fmt.Sprintf("📋 #%d %s", taskID, title)
	detail := joinContext(card.Board, card.Status, withDeadline(card.Deadline))
	if detail == "" {
		return head
	}
	return head + "\n" + detail
}

func formatTaskStatus(taskID int64, title, prev, status string, card taskCard, actor string) string {
	move := status
	if prev != "" && prev != status {
		move = prev + " → " + status
	}
	head := fmt.Sprintf("🔄 #%d %s: %s", taskID, title, move)
	detail := joinContext(card.Board, actor)
	if detail == "" {
		return head
	}
	return head + "\n" + detail
}

func formatTaskAssigned(taskID int64, title string, card taskCard) string {
	head := fmt.Sprintf("👤 Вам назначена задача #%d: %s", taskID, title)
	detail := joinContext(card.Board, card.Status, withDeadline(card.Deadline))
	if detail == "" {
		return head
	}
	return head + "\n" + detail
}

func formatTaskDeadline(taskID int64, title string, card taskCard) string {
	head := fmt.Sprintf("⏰ Дедлайн сегодня: #%d %s", taskID, title)
	detail := joinContext(card.Board, card.Status)
	if detail == "" {
		return head
	}
	return head + "\n" + detail
}

func withDeadline(deadline string) string {
	if deadline == "" {
		return ""
	}
	return "до " + deadline
}

// taskActionButtons attaches take/done buttons moving the task forward.
// Chat-only: dialog recipients cannot resolve a chat for callbacks.
func taskActionButtons(taskID int64) []any {
	return []any{InlineKeyboard([][]Button{
		{CallbackButton("Взять", fmt.Sprintf("take:%d", taskID)), CallbackButton("Готово", fmt.Sprintf("done:%d", taskID))},
	})}
}

func (n *Notifier) EnqueueTaskCreated(ctx context.Context, boardID, taskID int64, title string) error {
	chats, err := boardChatIDs(ctx, n.Pool, boardID)
	if err != nil {
		return err
	}
	card, err := n.loadTaskCard(ctx, taskID)
	if err != nil {
		return err
	}
	for _, chatID := range chats {
		chat := chatID
		if _, err := n.Enqueue(ctx, fmt.Sprintf("task-created:%d:%d", taskID, chat), NotificationDestination{ChatID: &chat}, OutgoingMessage{
			Text:        formatTaskCreated(taskID, title, card),
			Attachments: taskActionButtons(taskID),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (n *Notifier) EnqueueTaskStatus(ctx context.Context, boardID, taskID, columnID int64, title, status string, prevColumnID, actorID int64) error {
	chats, err := boardChatIDs(ctx, n.Pool, boardID)
	if err != nil {
		return err
	}
	card, err := n.loadTaskCard(ctx, taskID)
	if err != nil {
		return err
	}
	if card.Status == "" {
		card.Status = status
	}
	actor := n.memberDisplayName(ctx, boardID, actorID)
	text := formatTaskStatus(taskID, title, n.columnName(ctx, prevColumnID), status, card, actor)
	for _, chatID := range chats {
		chat := chatID
		if _, err := n.Enqueue(ctx, fmt.Sprintf("task-status:%d:%d:%d:%d", taskID, columnID, chat, n.Now().UnixNano()), NotificationDestination{ChatID: &chat}, OutgoingMessage{
			Text:        text,
			Attachments: taskActionButtons(taskID),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (n *Notifier) EnqueueTaskAssigned(ctx context.Context, taskID, userID int64, title string) error {
	card, err := n.loadTaskCard(ctx, taskID)
	if err != nil {
		return err
	}
	user := userID
	_, err = n.Enqueue(ctx, fmt.Sprintf("task-assigned:%d:%d", taskID, userID), NotificationDestination{UserID: &user}, OutgoingMessage{
		Text: formatTaskAssigned(taskID, title, card),
	})
	return err
}

func (n *Notifier) EnqueueDueDeadlines(ctx context.Context, date time.Time) error {
	day := date.Format("2006-01-02")
	rows, err := n.Pool.Query(ctx, `
		SELECT t.id, t.title, a.user_id
		FROM tasks t
		JOIN task_assignees a ON a.task_id = t.id
		WHERE t.deadline = $1::date
	`, day)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var taskID, userID int64
		var title string
		if err := rows.Scan(&taskID, &title, &userID); err != nil {
			return err
		}
		card, err := n.loadTaskCard(ctx, taskID)
		if err != nil {
			return err
		}
		user := userID
		if _, err := n.Enqueue(ctx, fmt.Sprintf("task-deadline:%d:%d:%s", taskID, userID, day), NotificationDestination{UserID: &user}, OutgoingMessage{
			Text: formatTaskDeadline(taskID, title, card),
		}); err != nil {
			return err
		}
	}
	return rows.Err()
}

func boardChatIDs(ctx context.Context, pool *pgxpool.Pool, boardID int64) ([]int64, error) {
	rows, err := pool.Query(ctx, `SELECT chat_id FROM board_chats WHERE board_id = $1 ORDER BY chat_id`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		ids = append(ids, chatID)
	}
	return ids, rows.Err()
}
