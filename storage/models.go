package storage

import (
	"time"
)

type ID string

type Base struct {
	Id        ID         `json:"id"`
	CreatedAt *time.Time `json:"created_at"`
}

type Message struct {
	Base
	Role       string `json:"role"`
	Content    string `json:"content"`
	Cost       int    `json:"cost"`
	Embeddings []float32
}

type Document struct {
	Base
	Content  string            `json:"content"`
	Filename string            `json:"filename,omitempty"`
	Source   string            `json:"source,omitempty"`
	FileHash string            `json:"filehash,omitempty"`
	Metadata map[string]string `json:"metadata"`
}

type Embeddings struct {
	Base
	DocumentId ID     `json:"document_id"`
	Content    string `json:"content"`
	Embeddings []float32
	Distance   float64
}

type MatchingDocument struct {
	ChunkContent string `json:"chunk_content"`
	Document     Document
	Distance     float64
}
