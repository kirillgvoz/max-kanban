package services

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxAPIBase = "https://platform-api2.max.ru"

type MaxBot struct {
	Token       string
	BotName     string
	FrontendURL string
	BaseURL     string
	HTTPClient  *http.Client
}

type Button struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Payload   string `json:"payload,omitempty"`
	URL       string `json:"url,omitempty"`
	WebApp    string `json:"web_app,omitempty"`
	ContactID *int64 `json:"contact_id,omitempty"`
}

type KeyboardAttachment struct {
	Type    string `json:"type"`
	Payload struct {
		Buttons [][]Button `json:"buttons"`
	} `json:"payload"`
}

type OutgoingMessage struct {
	Text        string `json:"text"`
	Attachments []any  `json:"attachments,omitempty"`
	Notify      *bool  `json:"notify,omitempty"`
	Format      string `json:"format,omitempty"`
}

type Answer struct {
	Message *OutgoingMessage `json:"message"`
}

type BotCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Subscription struct {
	URL         string   `json:"url"`
	UpdateTypes []string `json:"update_types"`
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("max api status %d: %s", e.Status, e.Body)
}

func NewMaxBot(token, botName, frontendURL string) *MaxBot {
	return &MaxBot{
		Token:       token,
		BotName:     botName,
		FrontendURL: frontendURL,
		BaseURL:     maxAPIBase,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *MaxBot) SetBaseURL(baseURL string) {
	b.BaseURL = baseURL
}

func (b *MaxBot) SetHTTPClient(client *http.Client) {
	b.HTTPClient = client
}

func LoadCustomCA(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no valid certificates in %s", path)
	}
	return pool, nil
}

func CallbackButton(text, payload string) Button {
	return Button{Type: "callback", Text: text, Payload: payload}
}

func LinkButton(text, rawURL string) Button {
	return Button{Type: "link", Text: text, URL: rawURL}
}

func OpenAppButton(text, webApp string) Button {
	return Button{Type: "open_app", Text: text, WebApp: webApp}
}

func InlineKeyboard(buttons [][]Button) KeyboardAttachment {
	attachment := KeyboardAttachment{Type: "inline_keyboard"}
	attachment.Payload.Buttons = buttons
	return attachment
}

func (b *MaxBot) SendWelcome(ctx context.Context, chatID int64) error {
	keyboard := InlineKeyboard([][]Button{
		{OpenAppButton("📋 Открыть TaskFlow", b.FrontendURL)},
		{LinkButton("📖 Помощь", "https://max.ru")},
	})
	return b.SendChatMessage(ctx, chatID, OutgoingMessage{
		Text:        "👋 Добро пожаловать в TaskFlow!\n\nУправляйте задачами прямо из мессенджера.\nСоздавайте организации, доски и работайте с командой.",
		Attachments: []any{keyboard},
	})
}

func (b *MaxBot) SendNotification(ctx context.Context, destination NotificationDestination, message OutgoingMessage) error {
	if destination.ChatID != nil {
		return b.SendChatMessage(ctx, *destination.ChatID, message)
	}
	if destination.UserID != nil {
		return b.SendDirectMessage(ctx, *destination.UserID, message)
	}
	return fmt.Errorf("notification destination is required")
}

func (b *MaxBot) SendChatMessage(ctx context.Context, chatID int64, message OutgoingMessage) error {
	return b.sendMessage(ctx, url.Values{"chat_id": {strconv.FormatInt(chatID, 10)}}, message)
}

func (b *MaxBot) SendDirectMessage(ctx context.Context, userID int64, message OutgoingMessage) error {
	return b.sendMessage(ctx, url.Values{"user_id": {strconv.FormatInt(userID, 10)}}, message)
}

func (b *MaxBot) AnswerCallback(ctx context.Context, callbackID string, message *OutgoingMessage) error {
	_, err := b.do(ctx, http.MethodPost, "/answers", url.Values{"callback_id": {callbackID}}, Answer{Message: message})
	return err
}

func (b *MaxBot) RegisterCommands(ctx context.Context) error {
	commands := []BotCommand{
		{Name: "start", Description: "Запустить TaskFlow"},
		{Name: "tasks", Description: "Мои задачи"},
		{Name: "new", Description: "Новая задача"},
		{Name: "link", Description: "Привязать доску к чату"},
		{Name: "unlink", Description: "Отвязать доску от чата"},
	}
	_, err := b.do(ctx, http.MethodPatch, "/me/commands", nil, map[string]any{"commands": commands})
	return err
}

func (b *MaxBot) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	body, err := b.do(ctx, http.MethodGet, "/subscriptions", nil, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Subscriptions []Subscription `json:"subscriptions"`
		Success       *bool          `json:"success"`
		Message       *string        `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if response.Success != nil && !*response.Success {
		message := ""
		if response.Message != nil {
			message = *response.Message
		}
		return nil, fmt.Errorf("list subscriptions failed: %s", message)
	}
	return response.Subscriptions, nil
}

func (b *MaxBot) EnsureSubscription(ctx context.Context, webhookURL string, updateTypes []string, secret string) error {
	subscriptions, err := b.ListSubscriptions(ctx)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		if subscription.URL == webhookURL {
			return nil
		}
	}
	body, err := b.do(ctx, http.MethodPost, "/subscriptions", nil, map[string]any{
		"url":          webhookURL,
		"update_types": updateTypes,
		"secret":       secret,
	})
	if err != nil {
		return err
	}
	var response struct {
		Success *bool   `json:"success"`
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	if response.Success != nil && !*response.Success {
		message := ""
		if response.Message != nil {
			message = *response.Message
		}
		return fmt.Errorf("create subscription failed: %s", message)
	}
	return nil
}

func (b *MaxBot) BuildDeepLink(path string) string {
	return fmt.Sprintf("https://max.ru/%s?startapp=%s", b.BotName, url.QueryEscape(path))
}

func (b *MaxBot) sendMessage(ctx context.Context, query url.Values, message OutgoingMessage) error {
	_, err := b.do(ctx, http.MethodPost, "/messages", query, message)
	if err == nil {
		return nil
	}
	// An open_app button references the mini-app URL attached to the bot in
	// the partner panel. If it is missing or does not match exactly, MAX
	// rejects the whole message — but the text itself is still valuable, so
	// retry once without the mini-app button instead of staying silent.
	stripped, ok := stripOpenAppButtons(message)
	if !ok || !isMiniAppLinkError(err) {
		return err
	}
	log.Printf("max bot: open_app button rejected (%v), retrying without the mini-app button", err)
	_, retryErr := b.do(ctx, http.MethodPost, "/messages", query, stripped)
	return retryErr
}

// isMiniAppLinkError reports MAX rejections caused by an unattached or
// mismatched mini-app URL in an open_app button: 404 "Link not found" and
// 400 "Field 'webApp' cannot be null".
func isMiniAppLinkError(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	body := strings.ToLower(apiErr.Body)
	switch apiErr.Status {
	case http.StatusNotFound:
		return strings.Contains(body, "link")
	case http.StatusBadRequest:
		return strings.Contains(body, "webapp") || strings.Contains(body, "web_app")
	default:
		return false
	}
}

// stripOpenAppButtons removes open_app buttons from an outgoing message,
// dropping keyboard rows and attachments left empty. It reports whether the
// message actually contained such buttons.
func stripOpenAppButtons(message OutgoingMessage) (OutgoingMessage, bool) {
	stripped := false
	attachments := make([]any, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		var keyboard *KeyboardAttachment
		var buttons [][]Button
		switch typed := attachment.(type) {
		case KeyboardAttachment:
			keyboard = &typed
			buttons = typed.Payload.Buttons
		case *KeyboardAttachment:
			keyboard = typed
			buttons = typed.Payload.Buttons
		default:
			attachments = append(attachments, attachment)
			continue
		}
		kept := make([][]Button, 0, len(buttons))
		for _, row := range buttons {
			keptRow := make([]Button, 0, len(row))
			for _, button := range row {
				if button.Type == "open_app" {
					stripped = true
					continue
				}
				keptRow = append(keptRow, button)
			}
			if len(keptRow) > 0 {
				kept = append(kept, keptRow)
			}
		}
		if len(kept) == 0 {
			continue
		}
		keyboard.Payload.Buttons = kept
		attachments = append(attachments, *keyboard)
	}
	if !stripped {
		return message, false
	}
	message.Attachments = attachments
	return message, true
}

func (b *MaxBot) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	endpoint := b.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", b.Token)
	client := b.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, &APIError{Status: response.StatusCode, Body: string(responseBody)}
	}
	return responseBody, nil
}
