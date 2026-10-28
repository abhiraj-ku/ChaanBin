package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// task prefix required by nomic-emebed-text emebeding
const (
	PrefixDocument = "search_document: "
	PrefixQuery    = "search_query: "
)

type OllamaClient struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// prepends document prefix (used during ingestion)
func (c *OllamaClient) GetDocPrefix(ctx context.Context, text string) ([]float32, error) {
	return c.GetEmbeds(ctx, PrefixDocument+text)

}

// prepends query prefix (used during ingestion)
func (c *OllamaClient) GetQueryPrefix(ctx context.Context, text string) ([]float32, error) {
	return c.GetEmbeds(ctx, PrefixQuery+text)

}

func (c *OllamaClient) GetEmbeds(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(ollamaRequest{
		Model:  c.Model,
		Prompt: text,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to encode ollams payload: %w", err)
	}

	endPoint := fmt.Sprintf("%s/api/embeddings", c.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endPoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama request failed with non 200 code: %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return result.Embedding, nil
}
