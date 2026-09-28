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
	if len(commands) != 3 {
		t.Fatalf("commands = %#v", body["commands"])
	}
	first, _ := commands[0].(map[string]any)
	if first["name"] != "start" || first["description"] == "" {
		t.Fatalf("command = %#v", first)
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
