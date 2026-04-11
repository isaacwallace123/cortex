package domain

// GenerationRequest is what callers send to the inference layer.
type GenerationRequest struct {
	Model       string
	Prompt      string
	Temperature float32
	MaxTokens   int32
}

// GenerationResponse is what the LLM provider returns.
type GenerationResponse struct {
	Text         string
	Model        string
	InputTokens  int32
	OutputTokens int32
}

// GenerationChunk is a single streamed token from CompleteStream.
// Done=true on the final chunk, which carries token counts.
type GenerationChunk struct {
	Token        string
	Done         bool
	Model        string
	InputTokens  int32
	OutputTokens int32
}

// Model describes an available LLM.
type Model struct {
	Name     string
	Size     string
	Modified string
}
