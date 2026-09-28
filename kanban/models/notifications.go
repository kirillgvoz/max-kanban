package models

import "time"

type BoardChat struct {
	ID        int64     `json:"id"`
	BoardID   int64     `json:"board_id"`
	ChatID    int64     `json:"chat_id"`
	Title     string    `json:"title,omitempty"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type BoardChatCreate struct {
	ChatID int64  `json:"chat_id" binding:"required"`
	Title  string `json:"title" binding:"max=200"`
}

type Notification struct {
	ID            int64     `json:"id"`
	EventKey      string    `json:"event_key"`
	ChatID        *int64    `json:"chat_id,omitempty"`
	UserID        *int64    `json:"user_id,omitempty"`
	Text          string    `json:"text"`
	Status        string    `json:"status"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	CreatedAt     time.Time `json:"created_at"`
}
