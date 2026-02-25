package storage

import (
	"time"
)

type ID string

type Message struct {
	Id         ID         `json:"id"`
	CreatedAt  *time.Time `json:"created_at"`
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Cost       int        `json:"cost"`
	Embeddings *[]float32
	Pinned     bool // when true the message stays in the conversation, never summarised nor removed
}
