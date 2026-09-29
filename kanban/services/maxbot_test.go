package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type recordedRequest struct {
	method string
	path   string
	query  string
	auth   string
	body   map[string]any
}

func TestSendChatMessageUsesOfficialContract(t *testing.T) {
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = json.Unmarshal(body, &decoded)
		requests = append(requests, recordedRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, auth: r.Header.Get("Authorization"), body: decoded})
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	if err := bot.SendChatMessage(context.Background(), 123, OutgoingMessage{
		Text: "Задача готова",
		Attachments: []any{InlineKeyboard([][]Button{
			{CallbackButton("Готово", "done:1")},
			{OpenAppButton("Открыть", "https://example.com/app")},
		})},
	}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].method != "POST" || requests[0].path != "/messages" || requests[0].query != "chat_id=123" || requests[0].auth != "token" {
		t.Fatalf("request = %#v", requests)
	}
	if requests[0].body["text"] != "Задача готова" || requests[0].body["chat_id"] != nil {
		t.Fatalf("body = %#v", requests[0].body)
	}
	attachments, _ := requests[0].body["attachments"].([]any)
	if len(attachments) != 1 {
		t.Fatalf("attachments = %#v", requests[0].body["attachments"])
	}
	keyboard, _ := attachments[0].(map[string]any)
	if keyboard["type"] != "inline_keyboard" {
		t.Fatalf("keyboard = %#v", keyboard)
	}
}

func TestSendChatMessageRetriesWithoutOpenAppOnLinkError(t *testing.T) {
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = json.Unmarshal(body, &decoded)
		requests = append(requests, recordedRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, auth: r.Header.Get("Authorization"), body: decoded})
		if len(requests) == 1 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"link.not.found","message":"Link not found"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	if err := bot.SendChatMessage(context.Background(), 123, OutgoingMessage{
		Text: "Доска привязана",
		Attachments: []any{InlineKeyboard([][]Button{
			{OpenAppButton("Открыть", "https://example.com/app")},
			{CallbackButton("Взять", "take:1")},
		})},
	}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	retryed, _ := requests[1].body["attachments"].([]any)
	if len(retryed) != 1 {
		t.Fatalf("retried attachments = %#v", requests[1].body["attachments"])
	}
	keyboard, _ := retryed[0].(map[string]any)
	payload, _ := keyboard["payload"].(map[string]any)
	rows, _ := payload["buttons"].([]any)
	if len(rows) != 1 {
		t.Fatalf("retried buttons = %#v", payload["buttons"])
	}
	row, _ := rows[0].([]any)
	if len(row) != 1 {
		t.Fatalf("retried row = %#v", rows[0])
	}
	button, _ := row[0].(map[string]any)
	if button["type"] != "callback" || button["payload"] != "take:1" {
		t.Fatalf("retried button = %#v", row[0])
	}
	if requests[1].body["text"] != "Доска привязана" {
		t.Fatalf("retried text = %#v", requests[1].body["text"])
	}
}

func TestSendChatMessageNoRetryOnChatNotFound(t *testing.T) {
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = json.Unmarshal(body, &decoded)
		requests = append(requests, recordedRequest{body: decoded})
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"chat.not.found","message":"Chat not found"}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	err := bot.SendChatMessage(context.Background(), 99999, OutgoingMessage{
		Text:        "Привет",
		Attachments: []any{InlineKeyboard([][]Button{{OpenAppButton("Открыть", "https://example.com/app")}})},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
}

func TestStripOpenAppButtonsKeepsTextOnlyWhenAlone(t *testing.T) {
	stripped, ok := stripOpenAppButtons(OutgoingMessage{
		Text:        "Привет",
		Attachments: []any{InlineKeyboard([][]Button{{OpenAppButton("Открыть", "https://example.com/app")}})},
	})
	if !ok {
		t.Fatal("expected buttons to be stripped")
	}
	if stripped.Text != "Привет" || len(stripped.Attachments) != 0 {
		t.Fatalf("stripped = %#v", stripped)
	}
	if _, ok := stripOpenAppButtons(OutgoingMessage{Text: "Привет"}); ok {
		t.Fatal("unexpected strip without open_app buttons")
	}
}

func TestBuildDeepLink(t *testing.T) {
	bot := NewMaxBot("token", "mybot", "https://example.com/app")
	if got := bot.BuildDeepLink("board_12"); got != "https://max.ru/mybot?startapp=board_12" {
		t.Fatalf("deep link = %q", got)
	}
	if got := bot.BuildDeepLink("board/12 ..\u2713"); got != "https://max.ru/mybot?startapp=board12" {
		t.Fatalf("sanitized deep link = %q", got)
	}
	if got := bot.BuildDeepLink("..."); got != "" {
		t.Fatalf("empty payload deep link = %q", got)
	}
	if got := bot.BuildAppLink(); got != "https://max.ru/mybot?startapp" {
		t.Fatalf("app link = %q", got)
	}
	nameless := NewMaxBot("token", "", "https://example.com/app")
	if got := nameless.BuildDeepLink("board_1"); got != "" {
		t.Fatalf("nameless deep link = %q", got)
	}
	if got := nameless.BuildAppLink(); got != "" {
		t.Fatalf("nameless app link = %q", got)
	}
	long := strings.Repeat("a", 600)
	if got := bot.BuildDeepLink(long); len(got) != len("https://max.ru/mybot?startapp=")+512 {
		t.Fatalf("capped deep link length = %d", len(got))
	}
}

func TestAnswerCallback(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	if err := bot.AnswerCallback(context.Background(), "callback-1", &OutgoingMessage{Text: "Принято"}); err != nil {
		t.Fatal(err)
	}
	if query != "callback_id=callback-1" {
		t.Fatalf("query = %q", query)
	}
}

func TestRegisterCommandsSendsOneRequest(t *testing.T) {
	var count int
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte(`{"commands":[]}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	if err := bot.RegisterCommands(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("requests = %d, want 1", count)
	}
	commands, _ := body["commands"].([]any)
	names := make([]string, 0, len(commands))
	for _, item := range commands {
		command, _ := item.(map[string]any)
		name, _ := command["name"].(string)
		description, _ := command["description"].(string)
		if name == "" || description == "" {
			t.Fatalf("command = %#v", item)
		}
		names = append(names, name)
	}
	for _, want := range []string{"start", "tasks", "new", "link", "unlink"} {
		found := false
		for _, name := range names {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("commands = %#v, want %q", body["commands"], want)
		}
	}
}

func TestEnsureSubscription(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"subscriptions":[]}`))
			return
		}
		posts++
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	bot := NewMaxBot("token", "bot", "https://example.com/app")
	bot.BaseURL = server.URL
	bot.HTTPClient = server.Client()
	if err := bot.EnsureSubscription(context.Background(), "https://example.com/webhook", []string{"message_created"}, "secret-value"); err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatalf("posts = %d, want 1", posts)
	}
}

func TestLoadCustomCA(t *testing.T) {
	if _, err := LoadCustomCA(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("LoadCustomCA accepted missing file")
	}
	path := filepath.Join(t.TempDir(), "ca.pem")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		t.Fatal(err)
	}
	pool, err := LoadCustomCA(path)
	if err != nil || pool == nil {
		t.Fatalf("LoadCustomCA error = %v", err)
	}
	if !strings.Contains(string(pemData), "BEGIN CERTIFICATE") {
		t.Fatal("test certificate malformed")
	}
}
