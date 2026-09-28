package services

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"kanban/db"
)

type recordingSender struct {
	messages []OutgoingMessage
	err      error
}

func (s *recordingSender) SendNotification(_ context.Context, _ NotificationDestination, message OutgoingMessage) error {
	if s.err != nil {
		return s.err
	}
	s.messages = append(s.messages, message)
	return nil
}

func prepareNotifierDatabase(t *testing.T) (context.Context, int64, int64, int64) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := db.Connect(ctx, os.Getenv("TEST_DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := db.RunMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"notification_outbox", "board_chats", "task_assignees", "comments", "checklist_items", "checklists", "tasks", "columns", "boards", "org_members", "organizations"} {
		if _, err := db.Pool.Exec(ctx, "TRUNCATE "+table+" RESTART IDENTITY CASCADE"); err != nil {
			t.Fatal(err)
		}
	}

	var orgID, boardID, taskID int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO organizations (name, slug, created_by) VALUES ('Org', 'org', 1) RETURNING id`).Scan(&orgID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO org_members (org_id, user_id, display_name, role) VALUES ($1, 1, 'Owner', 'owner')`, orgID); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `INSERT INTO boards (org_id, name, created_by) VALUES ($1, 'Board', 1) RETURNING id`, orgID).Scan(&boardID); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `INSERT INTO tasks (board_id, column_id, title, created_by) SELECT $1, id, 'Задача', 1 FROM columns WHERE board_id = $1 ORDER BY position LIMIT 1 RETURNING id`, boardID).Scan(&taskID); err != nil {
		// The shared schema may require explicit columns; seed one directly.
		if _, err := db.Pool.Exec(ctx, `INSERT INTO columns (board_id, name, position, color) VALUES ($1, 'Статус', 0, '#6366F1')`, boardID); err != nil {
			t.Fatal(err)
		}
		if err := db.Pool.QueryRow(ctx, `INSERT INTO tasks (board_id, column_id, title, created_by) SELECT $1, id, 'Задача', 1 FROM columns WHERE board_id = $1 ORDER BY position LIMIT 1 RETURNING id`, boardID).Scan(&taskID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO task_assignees (task_id, user_id, display_name) VALUES ($1, 1, 'Owner')`, taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO board_chats (board_id, chat_id, created_by) VALUES ($1, 77, 1)`, boardID); err != nil {
		t.Fatal(err)
	}
	return ctx, boardID, taskID, orgID
}

func TestNotifierEnqueueIsIdempotent(t *testing.T) {
	ctx, _, _, _ := prepareNotifierDatabase(t)
	notifier := NewNotifier(db.Pool, &recordingSender{})
	chat := int64(77)
	inserted, err := notifier.Enqueue(ctx, "event-1", NotificationDestination{ChatID: &chat}, OutgoingMessage{Text: "Привет"})
	if err != nil || !inserted {
		t.Fatalf("Enqueue() = %v, %v", inserted, err)
	}
	inserted, err = notifier.Enqueue(ctx, "event-1", NotificationDestination{ChatID: &chat}, OutgoingMessage{Text: "Привет"})
	if err != nil || inserted {
		t.Fatalf("duplicate Enqueue() = %v, %v", inserted, err)
	}
}

func TestNotifierProcessesChatMessages(t *testing.T) {
	ctx, boardID, taskID, _ := prepareNotifierDatabase(t)
	sender := &recordingSender{}
	notifier := NewNotifier(db.Pool, sender)
	notifier.RateLimit = 0
	if err := notifier.EnqueueTaskCreated(ctx, boardID, taskID, "Задача"); err != nil {
		t.Fatal(err)
	}
	if err := notifier.EnqueueTaskStatus(ctx, boardID, taskID, 1, "Задача", "В работе"); err != nil {
		t.Fatal(err)
	}
	if err := notifier.EnqueueTaskAssigned(ctx, taskID, 1, "Задача"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := notifier.ProcessOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.messages) != 3 {
		t.Fatalf("messages = %d, want 3", len(sender.messages))
	}
}

func TestNotifierRetriesFailures(t *testing.T) {
	ctx, _, _, _ := prepareNotifierDatabase(t)
	sender := &recordingSender{err: errors.New("max unavailable")}
	notifier := NewNotifier(db.Pool, sender)
	notifier.RateLimit = 0
	notifier.MaxRetries = 1
	chat := int64(77)
	if _, err := notifier.Enqueue(ctx, "failing", NotificationDestination{ChatID: &chat}, OutgoingMessage{Text: "Привет"}); err != nil {
		t.Fatal(err)
	}
	if err := notifier.ProcessOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	if err := db.Pool.QueryRow(ctx, `SELECT status, attempts FROM notification_outbox WHERE event_key = 'failing'`).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || attempts != 1 {
		t.Fatalf("status=%q attempts=%d, want failed/1", status, attempts)
	}
}

func TestNotifierDeadlines(t *testing.T) {
	ctx, _, taskID, _ := prepareNotifierDatabase(t)
	today := time.Now().Format("2006-01-02")
	if _, err := db.Pool.Exec(ctx, `UPDATE tasks SET deadline = $1 WHERE id = $2`, today, taskID); err != nil {
		t.Fatal(err)
	}
	notifier := NewNotifier(db.Pool, &recordingSender{})
	if err := notifier.EnqueueDueDeadlines(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE 'task-deadline:%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("deadline notifications = %d, want 1", count)
	}
	if err := notifier.EnqueueDueDeadlines(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE 'task-deadline:%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate deadline notifications = %d, want 1", count)
	}
}
