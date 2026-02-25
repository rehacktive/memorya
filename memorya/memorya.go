package storage

import (
	"fmt"
	"strings"

	"github.com/rehacktive/memorya/storage"
)

type Summarizer interface {
	Summarize(messages []storage.Message) (storage.Message, error)
}

type Memorya struct {
	storage          storage.Storage
	maxContextSize   int
	sequentialMemory []storage.Message
	summarizer       Summarizer
	pendingRecall    []float32
}

func InitMemorya(maxSize int, st storage.Storage) *Memorya {
	return &Memorya{
		storage:          st,
		maxContextSize:   maxSize,
		sequentialMemory: make([]storage.Message, 0),
	}
}

func InitMemoryaWithSummarizer(maxSize int, st storage.Storage, summarizer Summarizer) *Memorya {
	return &Memorya{
		storage:          st,
		maxContextSize:   maxSize,
		sequentialMemory: make([]storage.Message, 0),
		summarizer:       summarizer,
	}
}

func (m *Memorya) SetSummarizer(summarizer Summarizer) {
	m.summarizer = summarizer
}

func (m *Memorya) Reset() {
	m.sequentialMemory = make([]storage.Message, 0)
}

func (m *Memorya) AddMessage(message storage.Message, pinned bool) {
	fmt.Println("added message to memorya: " + message.Content)
	if message.Embeddings != nil && len(*message.Embeddings) > 0 {
		m.pendingRecall = append([]float32(nil), (*message.Embeddings)...)
	}

	err := m.storage.StoreMessage(message)
	if err != nil {
		panic(err)
	}
	// remove embeddings from memory
	message.Embeddings = nil
	message.Pinned = pinned
	m.sequentialMemory = append(m.sequentialMemory, message)

	// refresh memory
	m.Refresh()
}

func (m *Memorya) Refresh() {
	workingMemory := append([]storage.Message(nil), m.sequentialMemory...)
	if len(m.pendingRecall) > 0 {
		recalled := m.Remember(m.pendingRecall)
		if recallMessage, ok := buildRecallMessage(recalled, workingMemory); ok {
			workingMemory = append(workingMemory, recallMessage)
		}
		m.pendingRecall = nil
	}

	if m.maxContextSize <= 0 {
		m.sequentialMemory = workingMemory
		return
	}

	// Check if we've exceeded maxContextSize
	if len(workingMemory) <= m.maxContextSize {
		m.sequentialMemory = workingMemory
		return
	}

	pinned, unpinned := splitPinned(workingMemory)

	// Pinned messages can make the context exceed max size by design.
	available := m.maxContextSize - len(pinned)
	if available <= 0 {
		m.sequentialMemory = pinned
		return
	}

	if len(unpinned) <= available {
		m.sequentialMemory = mergeInOriginalOrder(workingMemory, pinned, unpinned)
		return
	}

	condensedUnpinned := unpinned[len(unpinned)-available:]
	if m.summarizer != nil && available > 1 {
		recentCount := available - 1
		summaryCandidates := unpinned[:len(unpinned)-recentCount]
		recent := unpinned[len(unpinned)-recentCount:]

		if len(summaryCandidates) > 0 {
			summary, err := m.summarizer.Summarize(summaryCandidates)
			if err == nil {
				summary.Pinned = false
				if summary.Role == "" {
					summary.Role = "system"
				}
				// Best effort persist of synthesized memory.
				_ = m.storage.StoreMessage(summary)
				condensedUnpinned = append([]storage.Message{summary}, recent...)
			}
		}
	}

	m.sequentialMemory = mergeInOriginalOrder(workingMemory, pinned, condensedUnpinned)
}

func (m *Memorya) GetMessages() []storage.Message {
	return m.sequentialMemory
}

// this will search on the database if there are related conversations via embeddings
func (m *Memorya) Remember(queryEmbeddings []float32) []storage.Message {
	if len(queryEmbeddings) == 0 {
		return nil
	}

	messages, err := m.storage.SearchRelatedMessages(queryEmbeddings)
	if err != nil {
		return nil
	}
	return messages
}

func splitPinned(messages []storage.Message) ([]storage.Message, []storage.Message) {
	pinned := make([]storage.Message, 0)
	unpinned := make([]storage.Message, 0)
	for _, msg := range messages {
		if msg.Pinned {
			pinned = append(pinned, msg)
			continue
		}
		unpinned = append(unpinned, msg)
	}
	return pinned, unpinned
}

func mergeInOriginalOrder(original []storage.Message, pinned []storage.Message, unpinned []storage.Message) []storage.Message {
	// Reconstruct a stable order that follows original pinned positions as much as possible.
	ret := make([]storage.Message, 0, len(pinned)+len(unpinned))

	pinnedIdx := 0
	unpinnedIdx := 0
	for _, msg := range original {
		if msg.Pinned {
			if pinnedIdx < len(pinned) {
				ret = append(ret, pinned[pinnedIdx])
				pinnedIdx++
			}
			continue
		}
		if unpinnedIdx < len(unpinned) {
			ret = append(ret, unpinned[unpinnedIdx])
			unpinnedIdx++
		}
	}

	for pinnedIdx < len(pinned) {
		ret = append(ret, pinned[pinnedIdx])
		pinnedIdx++
	}
	for unpinnedIdx < len(unpinned) {
		ret = append(ret, unpinned[unpinnedIdx])
		unpinnedIdx++
	}
	return ret
}

func buildRecallMessage(recalled []storage.Message, current []storage.Message) (storage.Message, bool) {
	if len(recalled) == 0 {
		return storage.Message{}, false
	}

	seen := make(map[string]bool, len(current))
	for _, msg := range current {
		key := msg.Role + "::" + msg.Content
		seen[key] = true
	}

	lines := make([]string, 0, 3)
	for _, msg := range recalled {
		if msg.Content == "" {
			continue
		}
		key := msg.Role + "::" + msg.Content
		if seen[key] {
			continue
		}
		role := msg.Role
		if role == "" {
			role = "unknown"
		}
		lines = append(lines, role+": "+msg.Content)
		if len(lines) == 3 {
			break
		}
	}

	if len(lines) == 0 {
		return storage.Message{}, false
	}

	return storage.Message{
		Role:    "system",
		Content: "Recalled context:\n- " + strings.Join(lines, "\n- "),
	}, true
}
