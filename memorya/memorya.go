package storage

import (
	"fmt"

	"github.com/rehacktive/memorya/storage"
)

type Memorya struct {
	storage          storage.Storage
	prompt           string
	maxContextSize   int
	sequentialMemory []storage.Message
}

func InitMemorya(maxSize int, storage storage.Storage) *Memorya {
	return &Memorya{
		storage:          storage,
		maxContextSize:   maxSize,
		sequentialMemory: make([]storage.Message, 0),
	}
}

func (m *Memorya) SetPrompt(p string) {
	m.prompt = p
}

func (m *Memorya) GetPrompt() string {
	return m.prompt
}

func (m *Memorya) Reset() {
	m.sequentialMemory = make([]storage.Message, 0)
}

func (m *Memorya) AddMessage(message storage.Message) {
	fmt.Println("added message to memorya: " + message.Content)

	err := m.storage.StoreMessage(message)
	if err != nil {
		panic(err)
	}
	// remove embeddings from memory
	message.Embeddings = []float32{}
	m.sequentialMemory = append(m.sequentialMemory, message)
}

func (m *Memorya) GetMessages() []storage.Message {
	return m.sequentialMemory
}

/*
see also: https://www.emergentmind.com/papers/2402.09727

making a "memory", similar to chatcontext, but working as short or long memory
short is the context itself, long is the database
with some rules:
short will become long
long will partially influence short (remembering, recalling...)
note: another policy in "context" could be vector distance, for each sentence calculate embeddings
and sort them by distance from the new question.

need for policies?

-> the conversation itself is stored as embeddings and each new question is translated into embeddings and a match function
   will return previous part of the conversation "related"

*/

// // this will search on the database if there are related conversations via embeddings
// func (m *Memorya) Remember(topic string) (ret string) {
// 	return
// }
