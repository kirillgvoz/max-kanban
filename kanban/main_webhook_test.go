package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"kanban/config"
	"kanban/db"
	"kanban/services"

	"github.com/gin-gonic/gin"
)

type fakeMaxCall struct {
	path  string
	query string
	body  map[string]any
}

type fakeMaxServer struct {
	calls []fakeMaxCall
}

func (f *fakeMaxServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var decoded map[string]any
	_ = json.Unmarshal(body, &decoded)
	f.calls = append(f.calls, fakeMaxCall{path: r.URL.Path, query: r.URL.RawQuery, body: decoded})
	w.Write([]byte(`{"success":true}`))
}

type fakeNotificationSender struct {
	messages []services.OutgoingMessage
}

func (f *fakeNotificationSender) SendNotification(_ context.Context, _ services.NotificationDestination, message services.OutgoingMessage) error {
	f.messages = append(f.messages, message)
	return nil
}

func prepareWebhookDatabase(t *testing.T) (context.Context, *httptest.Server, *fakeMaxServer, *fakeNotificationSender) {
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
	tables := []string{"notification_outbox", "webhook_events", "board_chats", "task_assignees", "comments", "checklist_items", "checklists", "tasks", "columns", "boards", "org_members", "organizations"}
	for _, table := range tables {
		if _, err := db.Pool.Exec(ctx, "TRUNCATE "+table+" RESTART IDENTITY CASCADE"); err != nil {
			t.Fatal(err)
		}
	}

	maxServer := &fakeMaxServer{}
	server := httptest.NewServer(http.HandlerFunc(maxServer.handler))
	t.Cleanup(server.Close)
	sender := &fakeNotificationSender{}
	services.SetDefaultNotifier(services.NewNotifier(db.Pool, sender))
	t.Cleanup(func() { services.SetDefaultNotifier(nil) })
	return ctx, server, maxServer, sender
}

func setupWebhookRouter(t *testing.T, maxServerURL string) (*httptest.Server, *services.MaxBot) {
	t.Helper()
	bot := services.NewMaxBot("token", "bot", "https://example.com/app")
	bot.SetBaseURL(maxServerURL)
	gin.SetMode(gin.TestMode)
	router := setupRouter(&config.Config{WebhookSecret: "webhook-secret"}, bot)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, bot
}

func postWebhook(t *testing.T, server *httptest.Server, payload map[string]any) int {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/webhook", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Max-Bot-Api-Secret", "webhook-secret")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	_, _ = io.ReadAll(response.Body)
	return response.StatusCode
}

func seedChatBoard(t *testing.T, ctx context.Context) (int64, int64, int64) {
	t.Helper()
	var orgID, boardID, firstColumn, taskID int64
	if err := db.Pool.QueryRow(ctx, `INSERT INTO organizations (name, slug, created_by) VALUES ('Org', 'org', 1) RETURNING id`).Scan(&orgID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO org_members (org_id, user_id, display_name, role) VALUES ($1, 1, 'Owner', 'owner')`, orgID); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `INSERT INTO boards (org_id, name, created_by) VALUES ($1, 'Board', 1) RETURNING id`, orgID).Scan(&boardID); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `INSERT INTO columns (board_id, name, position, color) VALUES ($1, 'Первая', 0, '#6366F1') RETURNING id`, boardID).Scan(&firstColumn); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO columns (board_id, name, position, color) VALUES ($1, 'Вторая', 1, '#F59E0B'), ($1, 'Третья', 2, '#10B981')`, boardID); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `INSERT INTO tasks (board_id, column_id, title, created_by) VALUES ($1, $2, 'Задача', 1) RETURNING id`, boardID, firstColumn).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO board_chats (board_id, chat_id, created_by) VALUES ($1, 77, 1)`, boardID); err != nil {
		t.Fatal(err)
	}
	return boardID, firstColumn, taskID
}

func countRows(t *testing.T, ctx context.Context, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.Pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func taskColumn(t *testing.T, ctx context.Context, taskID int64) int64 {
	t.Helper()
	var columnID int64
	if err := db.Pool.QueryRow(ctx, `SELECT column_id FROM tasks WHERE id = $1`, taskID).Scan(&columnID); err != nil {
		t.Fatal(err)
	}
	return columnID
}

func TestWebhookCallbackIsIdempotent(t *testing.T) {
	ctx, maxServer, maxCalls, _ := prepareWebhookDatabase(t)
	server, _ := setupWebhookRouter(t, maxServer.URL)
	_, _, taskID := seedChatBoard(t, ctx)
	payload := map[string]any{
		"update_type": "message_callback",
		"timestamp":   1780000000000,
		"callback": map[string]any{
			"callback_id": "callback-1",
			"payload":     fmt.Sprintf("done:%d", taskID),
			"user":        map[string]any{"user_id": 1},
			"message":     map[string]any{"recipient": map[string]any{"chat_id": 77}},
		},
	}

	if status := postWebhook(t, server, payload); status != 200 {
		t.Fatalf("first callback status = %d", status)
	}
	var finalColumn int64
	if err := db.Pool.QueryRow(ctx, `SELECT id FROM columns WHERE board_id = 1 ORDER BY position DESC LIMIT 1`).Scan(&finalColumn); err != nil {
		t.Fatal(err)
	}
	if got := taskColumn(t, ctx, taskID); got != finalColumn {
		t.Fatalf("task column = %d, want %d", got, finalColumn)
	}
	before := countRows(t, ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE 'task-status:%'`)
	if before != 1 {
		t.Fatalf("status notifications = %d, want 1", before)
	}
	if status := postWebhook(t, server, payload); status != 200 {
		t.Fatalf("duplicate callback status = %d", status)
	}
	if got := taskColumn(t, ctx, taskID); got != finalColumn {
		t.Fatalf("duplicate callback moved task to %d", got)
	}
	after := countRows(t, ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE 'task-status:%'`)
	if after != before {
		t.Fatalf("duplicate callback queued another notification")
	}
	answers := 0
	for _, call := range maxCalls.calls {
		if call.path == "/answers" && strings.Contains(call.query, "callback_id=callback-1") {
			answers++
		}
	}
	if answers != 2 {
		t.Fatalf("callback answers = %d, want 2", answers)
	}
}

func TestWebhookCallbackRejectsNonMember(t *testing.T) {
	ctx, maxServer, _, _ := prepareWebhookDatabase(t)
	server, _ := setupWebhookRouter(t, maxServer.URL)
	_, firstColumn, taskID := seedChatBoard(t, ctx)
	payload := map[string]any{
		"update_type": "message_callback",
		"timestamp":   1780000000001,
		"callback": map[string]any{
			"callback_id": "callback-2",
			"payload":     fmt.Sprintf("done:%d", taskID),
			"user":        map[string]any{"user_id": 99},
			"message":     map[string]any{"recipient": map[string]any{"chat_id": 77}},
		},
	}
	if status := postWebhook(t, server, payload); status != 200 {
		t.Fatalf("callback status = %d", status)
	}
	if got := taskColumn(t, ctx, taskID); got != firstColumn {
		t.Fatalf("non-member moved task to %d", got)
	}
	if count := countRows(t, ctx, `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE 'task-status:%'`); count != 0 {
		t.Fatalf("non-member queued %d notifications", count)
	}
}

func TestWebhookChatCommands(t *testing.T) {
	ctx, maxServer, maxCalls, _ := prepareWebhookDatabase(t)
	server, _ := setupWebhookRouter(t, maxServer.URL)
	boardID, _, _ := seedChatBoard(t, ctx)
	if _, err := db.Pool.Exec(ctx, `DELETE FROM board_chats WHERE board_id = $1`, boardID); err != nil {
		t.Fatal(err)
	}

	link := map[string]any{
		"update_type": "message_created",
		"timestamp":   1780000000002,
		"message": map[string]any{
			"body":      map[string]any{"text": fmt.Sprintf("/link board_%d", boardID)},
			"sender":    map[string]any{"user_id": 1},
			"recipient": map[string]any{"chat_id": 77},
		},
	}
	if status := postWebhook(t, server, link); status != 200 {
		t.Fatalf("link status = %d", status)
	}
	if count := countRows(t, ctx, `SELECT COUNT(*) FROM board_chats WHERE board_id = $1 AND chat_id = 77`, boardID); count != 1 {
		t.Fatalf("board chats = %d, want 1", count)
	}
	linkReply := false
	for _, call := range maxCalls.calls {
		if call.path != "/messages" || !strings.Contains(call.query, "chat_id=77") {
			continue
		}
		if text, _ := call.body["text"].(string); strings.Contains(text, fmt.Sprintf("https://max.ru/bot?startapp=board_%d", boardID)) {
			linkReply = true
		}
	}
	if !linkReply {
		t.Fatalf("link reply without board deep link: %#v", maxCalls.calls)
	}

	create := map[string]any{
		"update_type": "message_created",
		"timestamp":   1780000000003,
		"message": map[string]any{
			"body":      map[string]any{"text": "/new Позвонить клиенту"},
			"sender":    map[string]any{"user_id": 1},
			"recipient": map[string]any{"chat_id": 77},
		},
	}
	if status := postWebhook(t, server, create); status != 200 {
		t.Fatalf("create status = %d", status)
	}
	if count := countRows(t, ctx, `SELECT COUNT(*) FROM tasks WHERE board_id = $1 AND title = 'Позвонить клиенту'`, boardID); count != 1 {
		t.Fatalf("chat tasks = %d, want 1", count)
	}
}
