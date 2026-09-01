package providers

type Message struct {
	Role    string
	Content string
}
type ChatRequest struct {
	Model    string
	Messages []Message
}
type ChatResponse struct {
	Content string
}

type StreamChunk struct {
	Content string
}

type StreamHandler func(StreamChunk) error
