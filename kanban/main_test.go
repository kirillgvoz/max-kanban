package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"

	"kanban/config"
	"kanban/db"
	"kanban/models"
	"kanban/services"

	"github.com/gin-gonic/gin"
)

const integrationBotToken = "integration-token"

type integrationResponse struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

type stubNotificationSender struct{}

func (stubNotificationSender) SendNotification(_ context.Context, _ services.NotificationDestination, _ services.OutgoingMessage) error {
	return nil
}

func TestAPIEndToEnd(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := prepareIntegrationDatabase(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	services.SetDefaultNotifier(services.NewNotifier(db.Pool, stubNotificationSender{}))
	t.Cleanup(func() { services.SetDefaultNotifier(nil) })
	gin.SetMode(gin.TestMode)
	router := setupRouter(&config.Config{MaxBotToken: integrationBotToken, WebhookSecret: "secret"}, services.NewMaxBot("", "", ""))
	server := httptest.NewServer(router)
	defer server.Close()

	health := requestJSON(t, server.Client(), http.MethodGet, server.URL+"/max-kanban/api/health", nil, "")
	if health.Status != 200 || health.Body["status"] != "ok" || health.Body["database"] != "ok" {
		t.Fatalf("health status = %d body=%v", health.Status, health.Body)
	}
	anonymousMetrics := requestJSON(t, server.Client(), http.MethodGet, server.URL+"/api/metrics", nil, "")
	if anonymousMetrics.Status != 401 {
		t.Fatalf("anonymous metrics status=%d, want 401", anonymousMetrics.Status)
	}
	metrics := requestJSON(t, server.Client(), http.MethodGet, server.URL+"/api/metrics", nil, signedData(t, 1))
	if metrics.Status != 200 {
		t.Fatalf("metrics status=%d body=%v", metrics.Status, metrics.Body)
	}
	outbox, ok := metrics.Body["notification_outbox"].(map[string]any)
	if !ok || outbox["pending"] == nil || metrics.Body["websocket_clients"] == nil || metrics.Body["schema_migrations"] == nil {
		t.Fatalf("metrics body=%v", metrics.Body)
	}

	org := requestJSON(t, server.Client(), http.MethodPost, server.URL+"/api/orgs", map[string]any{"name": "Рабочая команда"}, signedData(t, 1))
	if org.Status != 201 {
		t.Fatalf("create org status = %d body=%v", org.Status, org.Body)
	}
	orgID := number(org.Body["id"])

	board := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/boards", server.URL, orgID), map[string]any{
		"name": "Проект",
		"columns": []map[string]any{
			{"name": "К выполнению", "color": "#6366F1"},
			{"name": "В работе", "color": "#F59E0B"},
			{"name": "Готово", "color": "#10B981"},
		},
	}, signedData(t, 1))
	if board.Status != 201 {
		t.Fatalf("create board status = %d body=%v", board.Status, board.Body)
	}
	boardID := number(board.Body["id"])

	archive := requestJSON(t, server.Client(), http.MethodPatch, fmt.Sprintf("%s/api/boards/%d", server.URL, boardID), map[string]any{"is_archived": true}, signedData(t, 1))
	if archive.Status != 200 {
		t.Fatalf("archive board status=%d body=%v", archive.Status, archive.Body)
	}
	activeBoards := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/orgs/%d/boards", server.URL, orgID), nil, signedData(t, 1))
	if activeBoards.Status != 200 {
		t.Fatalf("active boards status=%d", activeBoards.Status)
	}
	var activeList []map[string]any
	if err := json.Unmarshal(activeBoards.Raw, &activeList); err != nil || len(activeList) != 0 {
		t.Fatalf("active boards = %v, err=%v", activeBoards.Raw, err)
	}
	archivedBoards := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/orgs/%d/boards?include_archived=true", server.URL, orgID), nil, signedData(t, 1))
	if archivedBoards.Status != 200 || !strings.Contains(string(archivedBoards.Raw), `"is_archived":true`) {
		t.Fatalf("archived boards status=%d body=%v", archivedBoards.Status, archivedBoards.Body)
	}
	restore := requestJSON(t, server.Client(), http.MethodPatch, fmt.Sprintf("%s/api/boards/%d", server.URL, boardID), map[string]any{"is_archived": false}, signedData(t, 1))
	if restore.Status != 200 {
		t.Fatalf("restore board status=%d body=%v", restore.Status, restore.Body)
	}

	chat := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/boards/%d/chats", server.URL, boardID), map[string]any{"chat_id": 77, "title": "Рабочий чат"}, signedData(t, 1))
	if chat.Status != 201 || number(chat.Body["id"]) == 0 {
		t.Fatalf("create board chat status=%d body=%v", chat.Status, chat.Body)
	}
	chatID := number(chat.Body["id"])
	duplicateChat := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/boards/%d/chats", server.URL, boardID), map[string]any{"chat_id": 77, "title": "Обновлённый чат"}, signedData(t, 1))
	if duplicateChat.Status != 201 || number(duplicateChat.Body["id"]) != chatID || duplicateChat.Body["title"] != "Обновлённый чат" {
		t.Fatalf("duplicate board chat status=%d body=%v", duplicateChat.Status, duplicateChat.Body)
	}
	crossUserChats := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/boards/%d/chats", server.URL, boardID), nil, signedData(t, 99))
	if crossUserChats.Status != 404 {
		t.Fatalf("non-member board chats status=%d, want 404", crossUserChats.Status)
	}

	detail := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/boards/%d", server.URL, boardID), nil, signedData(t, 1))
	if detail.Status != 200 {
		t.Fatalf("board detail status=%d body=%v", detail.Status, detail.Body)
	}
	boardWithColumns := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/max-kanban/api/boards/%d", server.URL, boardID), nil, signedData(t, 1))
	if boardWithColumns.Status != 200 {
		t.Fatalf("aliased board detail status = %d", boardWithColumns.Status)
	}
	var boardDetail models.BoardDetail
	if err := json.Unmarshal(boardWithColumns.Raw, &boardDetail); err != nil {
		t.Fatal(err)
	}
	if len(boardDetail.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(boardDetail.Columns))
	}
	reversed := []int64{boardDetail.Columns[2].ID, boardDetail.Columns[1].ID, boardDetail.Columns[0].ID}
	reorder := requestJSON(t, server.Client(), http.MethodPatch, fmt.Sprintf("%s/api/boards/%d/columns/reorder", server.URL, boardID), map[string]any{"column_ids": reversed}, signedData(t, 1))
	if reorder.Status != 200 {
		t.Fatalf("reorder columns status=%d body=%v", reorder.Status, reorder.Body)
	}
	reorderedDetail := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/boards/%d", server.URL, boardID), nil, signedData(t, 1))
	var reordered models.BoardDetail
	if err := json.Unmarshal(reorderedDetail.Raw, &reordered); err != nil {
		t.Fatal(err)
	}
	for position, id := range reversed {
		if reordered.Columns[position].ID != id || reordered.Columns[position].Position != position {
			t.Fatalf("columns = %#v, want %#v in order", reordered.Columns, reversed)
		}
	}
	boardDetail = reordered

	task := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/boards/%d/tasks", server.URL, boardID), map[string]any{
		"title": "Подготовить отчёт", "description": "К пятнице", "column_id": boardDetail.Columns[0].ID, "priority": "high", "deadline": "2026-10-01",
	}, signedData(t, 1))
	if task.Status != 201 {
		t.Fatalf("create task status=%d body=%v", task.Status, task.Body)
	}
	taskID := number(task.Body["id"])
	if task.Body["deadline"] != "2026-10-01" {
		t.Fatalf("deadline = %v", task.Body["deadline"])
	}
	if got := outboxCount(t, "task-created:%"); got != 1 {
		t.Fatalf("task-created notifications = %d, want 1", got)
	}

	move := requestJSON(t, server.Client(), http.MethodPut, fmt.Sprintf("%s/api/tasks/%d/move", server.URL, taskID), map[string]any{"column_id": boardDetail.Columns[1].ID, "position": 0}, signedData(t, 1))
	if move.Status != 200 || number(move.Body["column_id"]) != boardDetail.Columns[1].ID || number(move.Body["position"]) != 0 {
		t.Fatalf("move task status=%d body=%v", move.Status, move.Body)
	}
	if got := outboxCount(t, "task-status:%"); got != 1 {
		t.Fatalf("task-status notifications = %d, want 1", got)
	}

	update := requestJSON(t, server.Client(), http.MethodPatch, fmt.Sprintf("%s/api/tasks/%d", server.URL, taskID), map[string]any{"deadline": nil}, signedData(t, 1))
	if update.Status != 200 || update.Body["deadline"] != nil {
		t.Fatalf("clear deadline status=%d body=%v", update.Status, update.Body)
	}

	checklist := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/tasks/%d/checklists", server.URL, taskID), map[string]any{"title": "Публикация"}, signedData(t, 1))
	if checklist.Status != 201 {
		t.Fatalf("create checklist status=%d body=%v", checklist.Status, checklist.Body)
	}
	checklistID := number(checklist.Body["id"])
	item := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/checklists/%d/items", server.URL, checklistID), map[string]any{"text": "Проверить цифры"}, signedData(t, 1))
	if item.Status != 201 {
		t.Fatalf("create item status=%d body=%v", item.Status, item.Body)
	}

	comment := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/tasks/%d/comments", server.URL, taskID), map[string]any{"text": "Готово"}, signedData(t, 1))
	if comment.Status != 201 || comment.Body["username"] != "tester1" {
		t.Fatalf("comment status=%d body=%v", comment.Status, comment.Body)
	}

	addMember := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/members", server.URL, orgID), map[string]any{"user_id": 2, "username": "tester2", "display_name": "Tester Two"}, signedData(t, 1))
	if addMember.Status != 200 {
		t.Fatalf("add member status=%d body=%v", addMember.Status, addMember.Body)
	}
	demoteOwner := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/members", server.URL, orgID), map[string]any{"user_id": 1, "role": "admin"}, signedData(t, 1))
	if demoteOwner.Status != 409 {
		t.Fatalf("demote last owner status=%d, want 409", demoteOwner.Status)
	}
	memberForbidden := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/members", server.URL, orgID), map[string]any{"user_id": 3}, signedData(t, 2))
	if memberForbidden.Status != 403 {
		t.Fatalf("member management status=%d, want 403", memberForbidden.Status)
	}
	promoteMember := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/members", server.URL, orgID), map[string]any{"user_id": 2, "role": "admin"}, signedData(t, 1))
	if promoteMember.Status != 200 {
		t.Fatalf("promote member status=%d body=%v", promoteMember.Status, promoteMember.Body)
	}
	orgResponse := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/orgs/%d", server.URL, orgID), nil, signedData(t, 1))
	if orgResponse.Status != 200 {
		t.Fatalf("org detail status=%d body=%v", orgResponse.Status, orgResponse.Body)
	}
	var organizationDetail models.OrgDetail
	if err := json.Unmarshal(orgResponse.Raw, &organizationDetail); err != nil {
		t.Fatal(err)
	}
	adminFound := false
	for _, member := range organizationDetail.Members {
		if member.UserID == 2 && member.Role == "admin" {
			adminFound = true
		}
	}
	if !adminFound {
		t.Fatalf("promoted member not found: %v", organizationDetail.Members)
	}
	assign := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/tasks/%d/assign", server.URL, taskID), map[string]any{"user_id": 2}, signedData(t, 1))
	if assign.Status != 200 {
		t.Fatalf("assign task status=%d body=%v", assign.Status, assign.Body)
	}
	if got := outboxCount(t, "task-assigned:%"); got != 1 {
		t.Fatalf("task-assigned notifications = %d, want 1", got)
	}
	addViewer := requestJSON(t, server.Client(), http.MethodPost, fmt.Sprintf("%s/api/orgs/%d/members", server.URL, orgID), map[string]any{"user_id": 3}, signedData(t, 1))
	if addViewer.Status != 200 {
		t.Fatalf("add viewer status=%d body=%v", addViewer.Status, addViewer.Body)
	}
	memberChats := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/boards/%d/chats", server.URL, boardID), nil, signedData(t, 3))
	if memberChats.Status != 403 {
		t.Fatalf("member board chats status=%d, want 403", memberChats.Status)
	}
	memberDeleteChat := requestJSON(t, server.Client(), http.MethodDelete, fmt.Sprintf("%s/api/board-chats/%d", server.URL, chatID), nil, signedData(t, 3))
	if memberDeleteChat.Status != 403 {
		t.Fatalf("member delete board chat status=%d, want 403", memberDeleteChat.Status)
	}
	deleteChat := requestJSON(t, server.Client(), http.MethodDelete, fmt.Sprintf("%s/api/board-chats/%d", server.URL, chatID), nil, signedData(t, 1))
	if deleteChat.Status != 200 {
		t.Fatalf("delete board chat status=%d body=%v", deleteChat.Status, deleteChat.Body)
	}
	missingChat := requestJSON(t, server.Client(), http.MethodDelete, fmt.Sprintf("%s/api/board-chats/%d", server.URL, chatID), nil, signedData(t, 1))
	if missingChat.Status != 404 {
		t.Fatalf("missing board chat status=%d, want 404", missingChat.Status)
	}
	otherBoard := requestJSON(t, server.Client(), http.MethodGet, fmt.Sprintf("%s/api/boards/%d", server.URL, boardID), nil, signedData(t, 4))
	if otherBoard.Status != 404 {
		t.Fatalf("non-member board status=%d, want 404", otherBoard.Status)
	}

	nonEmptyColumn := requestJSON(t, server.Client(), http.MethodDelete, fmt.Sprintf("%s/api/columns/%d", server.URL, boardDetail.Columns[1].ID), nil, signedData(t, 1))
	if nonEmptyColumn.Status != 409 {
		t.Fatalf("delete non-empty column status=%d, want 409", nonEmptyColumn.Status)
	}

	ticket := requestJSON(t, server.Client(), http.MethodPost, server.URL+"/api/ws-ticket", map[string]any{"org_id": orgID}, signedData(t, 1))
	if ticket.Status != 200 || ticket.Body["ticket"] == "" {
		t.Fatalf("websocket ticket status=%d body=%v", ticket.Status, ticket.Body)
	}

	deleteTask := requestJSON(t, server.Client(), http.MethodDelete, fmt.Sprintf("%s/api/tasks/%d", server.URL, taskID), nil, signedData(t, 1))
	if deleteTask.Status != 200 {
		t.Fatalf("delete task status=%d body=%v", deleteTask.Status, deleteTask.Body)
	}
}

func prepareIntegrationDatabase() error {
	if err := db.Connect(context.Background(), os.Getenv("TEST_DATABASE_URL")); err != nil {
		return err
	}
	if err := db.RunMigrations(context.Background()); err != nil {
		return err
	}
	_, err := db.Pool.Exec(context.Background(), `TRUNCATE notification_outbox, webhook_events, board_chats, task_assignees, comments, checklist_items, checklists, tasks, columns, boards, org_members, organizations RESTART IDENTITY CASCADE`)
	return err
}

func signedData(t *testing.T, userID int64) string {
	t.Helper()
	values := map[string]string{
		"query_id": fmt.Sprintf("q-%d", userID),
		"user":     fmt.Sprintf(`{"id":%d,"username":"tester%d","name":"Tester %d"}`, userID, userID, userID),
	}
	params := url.Values{}
	for key, value := range values {
		params.Set(key, value)
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(integrationBotToken))
	computed := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = computed.Write([]byte(strings.Join(parts, "\n")))
	params.Set("hash", hex.EncodeToString(computed.Sum(nil)))
	return params.Encode()
}

func requestJSON(t *testing.T, client *http.Client, method, endpoint string, body any, initData string) integrationResponse {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if initData != "" {
		req.Header.Set("X-Max-InitData", initData)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	response := integrationResponse{Status: res.StatusCode, Body: map[string]any{}, Raw: data}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &response.Body)
	}
	return response
}

func outboxCount(t *testing.T, pattern string) int {
	t.Helper()
	var count int
	if err := db.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM notification_outbox WHERE event_key LIKE $1`, pattern).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func number(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	default:
		return 0
	}
}
