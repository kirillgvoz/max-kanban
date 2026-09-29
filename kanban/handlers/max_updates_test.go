package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kanban/services"
)

func decodeUpdate(t *testing.T, raw string) MaxUpdate {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	var update MaxUpdate
	if err := decoder.Decode(&update); err != nil {
		t.Fatal(err)
	}
	return update
}

func TestDecodeMaxUpdatePreservesLargeIDs(t *testing.T) {
	update := decodeUpdate(t, `{
		"update_type": "message_callback",
		"timestamp": 1780000000000,
		"callback": {
			"callback_id": "callback-123",
			"payload": "take:9007199254740993",
			"user": {"user_id": 9007199254740993},
			"message": {"sender": {"user_id": 7}, "recipient": {"chat_id": "555"}, "body": {"text": "Задача"}}
		}
	}`)
	if update.Callback.CallbackID != "callback-123" {
		t.Fatalf("callback_id = %q", update.Callback.CallbackID)
	}
	if update.Callback.User.UserID.Int64() != 9007199254740993 {
		t.Fatalf("user_id = %d", update.Callback.User.UserID.Int64())
	}
	action, taskID, ok := maxCallbackAction(update.Callback.Payload)
	if !ok || action != "take" || taskID != 9007199254740993 {
		t.Fatalf("callback action = %q %d %v", action, taskID, ok)
	}
	chatID, ok := maxCallbackChat(update.Callback)
	if !ok || chatID != 555 {
		t.Fatalf("chat_id = %d %v", chatID, ok)
	}
	if key := webhookEventKey(update, []byte("raw")); key != "callback:callback-123" {
		t.Fatalf("event key = %q", key)
	}
}

func TestDecodeMaxMessage(t *testing.T) {
	update := decodeUpdate(t, `{
		"update_type": "message_created",
		"message": {"body": {"text": "/start"}, "sender": {"user_id": 42}, "recipient": {"chat_id": 77}}
	}`)
	if maxText(update.Message) != "/start" {
		t.Fatalf("text = %q", maxText(update.Message))
	}
	chatID, ok := maxMessageChat(update.Message)
	if !ok || chatID != 77 || update.Message.Sender.UserID.Int64() != 42 {
		t.Fatalf("message = chat %d %v sender %d", chatID, ok, update.Message.Sender.UserID.Int64())
	}
	if !strings.HasPrefix(webhookEventKey(update, []byte("raw")), "message_created:") {
		t.Fatalf("event key = %q", webhookEventKey(update, []byte("raw")))
	}
}

func TestOptionalMaxIDAcceptsNegativeGroupChats(t *testing.T) {
	update := decodeUpdate(t, `{
		"update_type": "bot_added",
		"timestamp": 1775025604499,
		"chat_id": -70801090403050,
		"user": {"user_id": 123456789}
	}`)
	chatID, ok := optionalMaxID(update.ChatID)
	if !ok || chatID != -70801090403050 {
		t.Fatalf("chat_id = %d %v, want -70801090403050 true", chatID, ok)
	}

	group := decodeUpdate(t, `{
		"update_type": "message_created",
		"message": {"body": {"text": "/start"}, "sender": {"user_id": 123456789}, "recipient": {"chat_id": -70801090403050, "chat_type": "chat"}}
	}`)
	groupChatID, ok := maxMessageChat(group.Message)
	if !ok || groupChatID != -70801090403050 {
		t.Fatalf("group chat_id = %d %v", groupChatID, ok)
	}

	if _, ok := optionalMaxID(nil); ok {
		t.Fatal("nil chat id accepted")
	}
	zero := MaxID(0)
	if _, ok := optionalMaxID(&zero); ok {
		t.Fatal("zero chat id accepted")
	}
}

func TestSendLinkCardNavigatesToBoard(t *testing.T) {
	var request struct {
		method string
		path   string
		query  string
		body   map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request.body)
		request.method = r.Method
		request.path = r.URL.Path
		request.query = r.URL.RawQuery
		w.Write([]byte(`{"message":{"body":{"mid":"mid.3"}}}`))
	}))
	defer server.Close()

	bot := services.NewMaxBot("token", "mybot", "https://example.com/app")
	bot.SetBaseURL(server.URL)
	bot.SetHTTPClient(server.Client())
	handler := NewWebhookHandler(bot, "secret")

	mid, err := handler.sendLinkCard(context.Background(), -77, "Доска «Маркетинг» привязана", "📋 Открыть доску", "https://max.ru/mybot?startapp=board_5")
	if err != nil {
		t.Fatal(err)
	}
	if mid != "mid.3" {
		t.Fatalf("mid = %q", mid)
	}
	if request.method != "POST" || request.path != "/messages" || request.query != "chat_id=-77" {
		t.Fatalf("request = %#v", request)
	}
	attachments, _ := request.body["attachments"].([]any)
	if len(attachments) != 1 {
		t.Fatalf("attachments = %#v", request.body["attachments"])
	}
	keyboard, _ := attachments[0].(map[string]any)
	payload, _ := keyboard["payload"].(map[string]any)
	rows, _ := payload["buttons"].([]any)
	row, _ := rows[0].([]any)
	button, _ := row[0].(map[string]any)
	if button["type"] != "link" || button["url"] != "https://max.ru/mybot?startapp=board_5" || button["text"] != "📋 Открыть доску" {
		t.Fatalf("button = %#v", row[0])
	}
	if _, present := button["web_app"]; present {
		t.Fatalf("link button must not carry web_app, got %#v", row[0])
	}
}

func TestParseChatCommand(t *testing.T) {
	cases := []struct {
		text    string
		command string
		args    string
	}{
		{"/start", "/start", ""},
		{"/tasks", "/tasks", ""},
		{"/start@se14445725_bot", "/start", ""},
		{"/link board_12", "/link", "board_12"},
		{"/link@se14445725_bot board_12", "/link", "board_12"},
		{"/unlink board_3", "/unlink", "board_3"},
		{"/new Позвонить клиенту", "/new", "Позвонить клиенту"},
		{"/new@se14445725_bot Позвонить клиенту", "/new", "Позвонить клиенту"},
		{"🦆", "🦆", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		command, args := parseChatCommand(tc.text)
		if command != tc.command || args != tc.args {
			t.Fatalf("parseChatCommand(%q) = (%q, %q), want (%q, %q)", tc.text, command, args, tc.command, tc.args)
		}
	}
}
