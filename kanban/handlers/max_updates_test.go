package handlers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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
