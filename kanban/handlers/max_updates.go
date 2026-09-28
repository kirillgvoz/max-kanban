package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type MaxID int64

func (id *MaxID) UnmarshalJSON(data []byte) error {
	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		parsed, err := number.Int64()
		if err != nil {
			return err
		}
		*id = MaxID(parsed)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	parsed, err := numberFromString(text)
	if err != nil {
		return err
	}
	*id = MaxID(parsed)
	return nil
}

func (id MaxID) Int64() int64 {
	return int64(id)
}

type MaxUser struct {
	UserID   MaxID  `json:"user_id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type MaxRecipient struct {
	ChatID   *MaxID `json:"chat_id"`
	UserID   *MaxID `json:"user_id"`
	ChatType string `json:"chat_type"`
}

type MaxMessageBody struct {
	Text string `json:"text"`
}

type MaxMessage struct {
	Body      MaxMessageBody `json:"body"`
	Sender    MaxUser        `json:"sender"`
	Recipient MaxRecipient   `json:"recipient"`
}

type MaxCallback struct {
	CallbackID string      `json:"callback_id"`
	Payload    string      `json:"payload"`
	User       MaxUser     `json:"user"`
	Message    *MaxMessage `json:"message"`
}

type MaxUpdate struct {
	UpdateType string       `json:"update_type"`
	Timestamp  MaxID        `json:"timestamp"`
	ChatID     *MaxID       `json:"chat_id"`
	User       MaxUser      `json:"user"`
	Message    *MaxMessage  `json:"message"`
	Callback   *MaxCallback `json:"callback"`
}

func webhookEventKey(update MaxUpdate, raw []byte) string {
	if update.Callback != nil && strings.TrimSpace(update.Callback.CallbackID) != "" {
		return "callback:" + strings.TrimSpace(update.Callback.CallbackID)
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%s:%d:%s", update.UpdateType, update.Timestamp.Int64(), hex.EncodeToString(sum[:]))
}

func optionalMaxID(value *MaxID) (int64, bool) {
	if value == nil || *value <= 0 {
		return 0, false
	}
	return value.Int64(), true
}

func numberFromString(text string) (int64, error) {
	var number json.Number = json.Number(strings.TrimSpace(text))
	return number.Int64()
}

func maxCallbackAction(payload string) (string, int64, bool) {
	for _, action := range []string{"take:", "done:"} {
		if !strings.HasPrefix(payload, action) {
			continue
		}
		var number json.Number = json.Number(strings.TrimPrefix(payload, action))
		id, err := number.Int64()
		if err != nil || id <= 0 {
			return "", 0, false
		}
		return strings.TrimSuffix(action, ":"), id, true
	}
	return "", 0, false
}

func maxText(message *MaxMessage) string {
	if message == nil {
		return ""
	}
	return strings.TrimSpace(message.Body.Text)
}

func maxMessageChat(message *MaxMessage) (int64, bool) {
	if message == nil {
		return 0, false
	}
	if chatID, ok := optionalMaxID(message.Recipient.ChatID); ok {
		return chatID, true
	}
	return 0, false
}

func maxCallbackChat(callback *MaxCallback) (int64, bool) {
	if callback == nil {
		return 0, false
	}
	if callback.Message != nil {
		if chatID, ok := maxMessageChat(callback.Message); ok {
			return chatID, true
		}
	}
	return 0, false
}
