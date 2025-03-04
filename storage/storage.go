package storage

type Storage interface {
	StoreMessage(message Message) error

	StoreDocument(document Document) (ID, error)

	StoreEmbeddings(embeddings Embeddings) error

	Search(query []float32) ([]MatchingDocument, error)
	SearchRelatedMessages(query []float32) ([]Message, error)
}
