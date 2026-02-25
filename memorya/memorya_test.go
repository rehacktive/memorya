package storage

import (
	"errors"
	"strings"
	"testing"

	st "github.com/rehacktive/memorya/storage"
)

type fakeStorage struct {
	messages []st.Message
	related  []st.Message
	queries  [][]float32
}

func (f *fakeStorage) StoreMessage(message st.Message) error {
	f.messages = append(f.messages, message)
	return nil
}

func (f *fakeStorage) SearchRelatedMessages(query []float32) ([]st.Message, error) {
	f.queries = append(f.queries, append([]float32(nil), query...))
	return f.related, nil
}

type fakeSummarizer struct {
	err      error
	captured []st.Message
}

func (f *fakeSummarizer) Summarize(messages []st.Message) (st.Message, error) {
	f.captured = append(f.captured[:0], messages...)
	if f.err != nil {
		return st.Message{}, f.err
	}

	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		parts = append(parts, msg.Content)
	}

	return st.Message{
		Role:    "system",
		Content: "summary: " + strings.Join(parts, " | "),
	}, nil
}

func TestRefreshPreservesPinnedMessages(t *testing.T) {
	store := &fakeStorage{}
	mem := InitMemorya(2, store)

	mem.AddMessage(st.Message{Role: "system", Content: "pin-1"}, true)
	mem.AddMessage(st.Message{Role: "user", Content: "normal"}, false)
	mem.AddMessage(st.Message{Role: "system", Content: "pin-2"}, true)

	msgs := mem.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !msgs[0].Pinned || msgs[0].Content != "pin-1" {
		t.Fatalf("expected first pinned message to stay, got %+v", msgs[0])
	}
	if !msgs[1].Pinned || msgs[1].Content != "pin-2" {
		t.Fatalf("expected second pinned message to stay, got %+v", msgs[1])
	}
}

func TestRefreshSummarizesOnlyUnpinnedWhenNeeded(t *testing.T) {
	store := &fakeStorage{}
	summarizer := &fakeSummarizer{}
	mem := InitMemoryaWithSummarizer(3, store, summarizer)

	mem.AddMessage(st.Message{Role: "user", Content: "a"}, false)
	mem.AddMessage(st.Message{Role: "system", Content: "pin"}, true)
	mem.AddMessage(st.Message{Role: "assistant", Content: "b"}, false)
	mem.AddMessage(st.Message{Role: "user", Content: "c"}, false)

	msgs := mem.GetMessages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "summary: a | b" {
		t.Fatalf("expected summary as first message, got %+v", msgs[0])
	}
	if !msgs[1].Pinned || msgs[1].Content != "pin" {
		t.Fatalf("expected pinned message untouched in middle, got %+v", msgs[1])
	}
	if msgs[2].Content != "c" {
		t.Fatalf("expected most recent unpinned to remain, got %+v", msgs[2])
	}

	if len(summarizer.captured) != 2 {
		t.Fatalf("expected summarizer to receive two unpinned messages, got %d", len(summarizer.captured))
	}
	for _, msg := range summarizer.captured {
		if msg.Pinned {
			t.Fatalf("summarizer should never receive pinned messages")
		}
	}
}

func TestRefreshFallsBackWhenSummarizerFails(t *testing.T) {
	store := &fakeStorage{}
	summarizer := &fakeSummarizer{err: errors.New("boom")}
	mem := InitMemoryaWithSummarizer(2, store, summarizer)

	mem.AddMessage(st.Message{Role: "user", Content: "a"}, false)
	mem.AddMessage(st.Message{Role: "user", Content: "b"}, false)
	mem.AddMessage(st.Message{Role: "user", Content: "c"}, false)

	msgs := mem.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "b" || msgs[1].Content != "c" {
		t.Fatalf("expected fallback to keep most recent messages, got %+v", msgs)
	}
}

func TestRefreshUsesRememberForEmbeddedMessage(t *testing.T) {
	store := &fakeStorage{
		related: []st.Message{
			{Role: "assistant", Content: "old relevant fact"},
			{Role: "user", Content: "secondary memory"},
		},
	}
	mem := InitMemorya(4, store)

	mem.AddMessage(st.Message{Role: "user", Content: "current question"}, false)
	emb := []float32{0.1, 0.2, 0.3}
	mem.AddMessage(st.Message{Role: "user", Content: "new question", Embeddings: &emb}, false)

	if len(store.queries) != 1 {
		t.Fatalf("expected one memory search call, got %d", len(store.queries))
	}
	if len(store.queries[0]) != 3 {
		t.Fatalf("expected embedding query of length 3, got %d", len(store.queries[0]))
	}

	msgs := mem.GetMessages()
	if len(msgs) != 3 {
		t.Fatalf("expected context to include recall message, got %d messages", len(msgs))
	}
	if msgs[2].Role != "system" || !strings.Contains(msgs[2].Content, "Recalled context:") {
		t.Fatalf("expected trailing system recall message, got %+v", msgs[2])
	}
}

func TestRefreshSkipsRecallDuplicates(t *testing.T) {
	store := &fakeStorage{
		related: []st.Message{
			{Role: "user", Content: "existing"},
		},
	}
	mem := InitMemorya(5, store)

	mem.AddMessage(st.Message{Role: "user", Content: "existing"}, false)
	emb := []float32{1}
	mem.AddMessage(st.Message{Role: "user", Content: "trigger", Embeddings: &emb}, false)

	msgs := mem.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected no additional recall message for duplicates, got %d", len(msgs))
	}
}

func TestAddMessageStoresMessageWhenEmbeddingsPresent(t *testing.T) {
	store := &fakeStorage{}
	mem := InitMemorya(5, store)

	emb := []float32{0.11, -0.2}
	mem.AddMessage(st.Message{
		Role:       "user",
		Content:    "embed me",
		Embeddings: &emb,
	}, false)

	if len(store.messages) != 1 {
		t.Fatalf("expected one stored message, got %d", len(store.messages))
	}
}
