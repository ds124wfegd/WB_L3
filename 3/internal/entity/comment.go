package entity

import (
	"encoding/json"
	"time"
)

type Comment struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Children  []Comment `json:"children,omitempty"`
}

type CreateCommentRequest struct {
	ParentID string `json:"parent_id"`
	Author   string `json:"author"`
	Text     string `json:"text"`
}

type CommentsResponse struct {
	Comments []Comment `json:"comments"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

type SearchRequest struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
}

// Для сериализации в Redis
func (c *Comment) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Comment) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
