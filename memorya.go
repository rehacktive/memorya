package memorya

type Memorya struct {
	maxContextSize int
}

func InitMemorya(maxSize int) *Memorya {
	return &Memorya{
		maxContextSize: maxSize,
	}
}
