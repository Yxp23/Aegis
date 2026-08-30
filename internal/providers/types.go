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
