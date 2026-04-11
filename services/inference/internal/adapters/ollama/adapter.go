package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
)

// Adapter calls the Ollama REST API.
// Ollama API reference: https://github.com/ollama/ollama/blob/main/docs/api.md
type Adapter struct {
	baseURL string
	client  *http.Client
}

func NewAdapter(baseURL string) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM completions can be slow
		},
	}
}

type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options generateOptions `json:"options,omitempty"`
}

type generateOptions struct {
	Temperature float32 `json:"temperature,omitempty"`
	NumPredict  int32   `json:"num_predict,omitempty"`
}

type generateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	PromptEvalCount    int32  `json:"prompt_eval_count"`
	EvalCount          int32  `json:"eval_count"`
}

func (a *Adapter) Complete(ctx context.Context, req domain.GenerationRequest) (domain.GenerationResponse, error) {
	body := generateRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		Stream: false,
		Options: generateOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	respBody, err := a.post(ctx, "/api/generate", body)
	if err != nil {
		return domain.GenerationResponse{}, fmt.Errorf("ollama generate: %w", err)
	}

	var genResp generateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return domain.GenerationResponse{}, fmt.Errorf("ollama response decode: %w", err)
	}

	return domain.GenerationResponse{
		Text:         genResp.Response,
		Model:        genResp.Model,
		InputTokens:  genResp.PromptEvalCount,
		OutputTokens: genResp.EvalCount,
	}, nil
}

// CompleteStream calls Ollama with stream:true and sends each token to the
// returned channel. The channel is closed when streaming finishes or errors.
// The final domain.GenerationChunk has Done=true and carries token counts.
func (a *Adapter) CompleteStream(ctx context.Context, req domain.GenerationRequest) (<-chan domain.GenerationChunk, error) {
	body := generateRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		Stream: true,
		Options: generateOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Use a streaming client (no global timeout â€” context cancels it).
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama stream: %w", err)
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama returned %d", resp.StatusCode)
	}

	ch := make(chan domain.GenerationChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var chunk generateResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- domain.GenerationChunk{
				Token:        chunk.Response,
				Done:         chunk.Done,
				Model:        chunk.Model,
				InputTokens:  chunk.PromptEvalCount,
				OutputTokens: chunk.EvalCount,
			}:
			}
			if chunk.Done {
				return
			}
		}
	}()
	return ch, nil
}

type tagsResponse struct {
	Models []tagModel `json:"models"`
}

type tagModel struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

func (a *Adapter) ListModels(ctx context.Context) ([]domain.Model, error) {
	respBody, err := a.get(ctx, "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("ollama list models: %w", err)
	}

	var tagsResp tagsResponse
	if err := json.Unmarshal(respBody, &tagsResp); err != nil {
		return nil, fmt.Errorf("ollama tags decode: %w", err)
	}

	models := make([]domain.Model, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		models = append(models, domain.Model{
			Name:     m.Name,
			Size:     fmt.Sprintf("%d", m.Size),
			Modified: m.ModifiedAt,
		})
	}
	return models, nil
}

func (a *Adapter) post(ctx context.Context, path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return a.do(req)
}

func (a *Adapter) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return a.do(req)
}

func (a *Adapter) do(req *http.Request) ([]byte, error) {
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %s: %w", req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

