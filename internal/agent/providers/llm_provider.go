package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// LLMProvider adalah abstraksi untuk berbagai LLM provider (Gemini, OpenAI, dll)
type LLMProvider interface {
	// GenerateText menghasilkan teks berdasarkan prompt
	GenerateText(ctx context.Context, prompt string) (string, error)

	// GenerateStructured menghasilkan output terstruktur (JSON)
	GenerateStructured(ctx context.Context, prompt string, output interface{}) error

	// Close menutup koneksi provider
	Close()
}

// GeminiProvider adalah implementasi LLMProvider untuk Google Gemini
type GeminiProvider struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiProvider membuat instance baru Gemini provider
func NewGeminiProvider(ctx context.Context, modelName string) (*GeminiProvider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Default model jika tidak dispesifikkan
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	model := client.GenerativeModel(modelName)
	model.SetTemperature(0.7)

	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

// GenerateText implements LLMProvider
func (g *GeminiProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no response from LLM")
	}

	// Extract text from first candidate
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			return string(txt), nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

// GenerateStructured implements LLMProvider for JSON output
func (g *GeminiProvider) GenerateStructured(ctx context.Context, prompt string, output interface{}) error {
	// Set response type to JSON
	originalMIME := g.model.ResponseMIMEType
	g.model.ResponseMIMEType = "application/json"
	defer func() {
		g.model.ResponseMIMEType = originalMIME
	}()

	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			if err := json.Unmarshal([]byte(txt), output); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("no text content in response")
}

// Close implements LLMProvider
func (g *GeminiProvider) Close() {
	g.client.Close()
}
