# Memorya

Memorya is a lightweight memory layer for LLM applications in Go.

It keeps an in-memory working context, persists messages to your storage backend, recalls related history through vector search, and compresses old context with an LLM summarizer when needed.

## Features

- Sequential conversation memory for active context (`GetMessages()`).
- Persistent storage integration (`StoreMessage` on each added message).
- Pinned messages that are never summarized or removed from active context.
- Automatic context shrinking when max size is reached.
- Optional LLM summarization hook for old non-pinned messages.
- Long-term recall via `SearchRelatedMessages` when new embedded messages are added.
- Duplicate-safe recalled context injection.

## Install

```bash
go get github.com/rehacktive/memorya
```

## Project Structure

- `memorya/`: memory orchestration logic.
- `storage/`: interfaces and data models.

## Core Types

### Storage interface

You provide an implementation of the `storage.Storage` interface:

```go
type Storage interface {
    StoreMessage(message Message) error
    SearchRelatedMessages(query []float32) ([]Message, error)
}
```

### Storage contract (important)

To make Memorya work correctly, your storage implementation must:

1. Persist the full message in `StoreMessage`.
2. If `message.Embeddings` is present, index that vector for similarity search (inside `StoreMessage`).
3. Implement `SearchRelatedMessages(query)` to return messages semantically close to `query`.
4. Return messages in best-first order (most relevant first), ideally excluding the exact current message if your backend can detect it.

### Message

`Message` supports:

- `Embeddings *[]float32`: optional query vector used for recall.
- `Pinned bool`: if true, message is never summarized or removed from active context.

## Quick Start

```go
package main

import (
    "fmt"

    mem "github.com/rehacktive/memorya/memorya"
    st "github.com/rehacktive/memorya/storage"
)

type MyStorage struct{}

func (s *MyStorage) StoreMessage(message st.Message) error {
    // 1) Persist the message in your primary store.
    // 2) If message.Embeddings != nil, also upsert vector index entry here.
    return nil
}
func (s *MyStorage) SearchRelatedMessages(query []float32) ([]st.Message, error) {
    // Run vector similarity search and map results to stored messages.
    return []st.Message{
        {Role: "assistant", Content: "Previously discussed deployment strategy."},
    }, nil
}

func main() {
    store := &MyStorage{}
    memory := mem.InitMemorya(12, store)

    // Normal message.
    memory.AddMessage(st.Message{
        Role:    "user",
        Content: "How should we deploy this service?",
    }, false)

    // Pinned message (never removed/summarized).
    memory.AddMessage(st.Message{
        Role:    "system",
        Content: "Always answer in concise bullet points.",
    }, true)

    // Embedded message triggers recall in Refresh().
    emb := []float32{0.12, 0.98, -0.44}
    memory.AddMessage(st.Message{
        Role:       "user",
        Content:    "Remind me what we decided last time.",
        Embeddings: &emb,
    }, false)

    for _, msg := range memory.GetMessages() {
        fmt.Println(msg.Role, msg.Content)
    }
}
```

## Optional LLM Summarization

Implement `Summarizer` and initialize with `InitMemoryaWithSummarizer(...)`.

```go
type MySummarizer struct{}

func (s *MySummarizer) Summarize(messages []st.Message) (st.Message, error) {
    return st.Message{
        Role:    "system",
        Content: "Summary of earlier context...",
    }, nil
}
```

```go
memory := mem.InitMemoryaWithSummarizer(12, store, &MySummarizer{})
```

Or set/replace later:

```go
memory.SetSummarizer(&MySummarizer{})
```

## Refresh Behavior

`AddMessage(...)` automatically calls `Refresh()`.

During refresh:

1. If the new message had embeddings, Memorya runs `Remember()` via `SearchRelatedMessages`.
2. Related messages are compacted into one system recall message and injected into active context (deduplicated).
3. If context exceeds max size:
   - pinned messages are always preserved,
   - non-pinned messages are trimmed,
   - if summarizer is configured, older non-pinned messages are summarized.

## API Summary

- `InitMemorya(maxSize int, st storage.Storage) *Memorya`
- `InitMemoryaWithSummarizer(maxSize int, st storage.Storage, summarizer Summarizer) *Memorya`
- `SetSummarizer(summarizer Summarizer)`
- `AddMessage(message storage.Message, pinned bool)`
- `GetMessages() []storage.Message`
- `Reset()`
- `Refresh()`
- `Remember(queryEmbeddings []float32) []storage.Message`
