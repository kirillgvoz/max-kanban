package models

type Column struct {
	ID       int64  `json:"id"`
	BoardID  int64  `json:"board_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Color    string `json:"color"`
}

type ColumnCreate struct {
	Name     string `json:"name" binding:"required,min=1,max=80"`
	Position *int   `json:"position,omitempty" binding:"omitempty,gte=0"`
	Color    string `json:"color" binding:"omitempty,len=7"`
}

type ColumnUpdate struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=1,max=80"`
	Color *string `json:"color,omitempty" binding:"omitempty,len=7"`
}

type ColumnReorder struct {
	ColumnIDs []int64 `json:"column_ids" binding:"required,min=1,max=30,dive,gt=0"`
}
