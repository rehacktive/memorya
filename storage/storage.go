package storage

type Storage interface {
	StoreMessage(message Message) error
	SearchRelatedMessages(query []float32) ([]Message, error)
}
