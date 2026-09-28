package models

import "time"

type Comment struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"created_at"`
}

type CommentCreate struct {
	Text string `json:"text" binding:"required,min=1,max=5000"`
}
