package models

type Checklist struct {
	ID       int64           `json:"id"`
	TaskID   int64           `json:"task_id"`
	Title    string          `json:"title"`
	Position int             `json:"position"`
	Items    []ChecklistItem `json:"items"`
}

type ChecklistItem struct {
	ID          int64  `json:"id"`
	ChecklistID int64  `json:"checklist_id"`
	Text        string `json:"text"`
	IsDone      bool   `json:"is_done"`
	Position    int    `json:"position"`
}

type ChecklistCreate struct {
	Title string `json:"title" binding:"omitempty,max=120"`
}

type ChecklistUpdate struct {
	Title *string `json:"title,omitempty" binding:"omitempty,min=1,max=120"`
}

type ChecklistItemCreate struct {
	Text string `json:"text" binding:"required,min=1,max=500"`
}

type ChecklistItemUpdate struct {
	Text   *string `json:"text,omitempty" binding:"omitempty,min=1,max=500"`
	IsDone *bool   `json:"is_done,omitempty"`
}
