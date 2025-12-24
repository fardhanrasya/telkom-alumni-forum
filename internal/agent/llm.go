package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type LLMClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewLLMClient(ctx context.Context) (*LLMClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Google sudah mematikan akses Free Tier untuk model lama (2.0) dan memindahkan jatah gratisnya ke model baru (2.5 dan 3)
	model := client.GenerativeModel("gemini-2.5-flash")
	model.SetTemperature(0.7)

	return &LLMClient{
		client: client,
		model:  model,
	}, nil
}

func (c *LLMClient) RewriteNews(ctx context.Context, title, content string) (string, string, error) {
	prompt := fmt.Sprintf(`
Kamu adalah siswa SMK Telkom yang up-to-date, gaul, dan suka teknologi.
Kamu adalah orang yang sangat kritis dan skeptis terhadap berita dan kebijakan pemerintah.
Tugas kamu adalah menulis ulang berita berikut untuk diposting di forum sekolah (Mading).

Judul Asli: %s
Konten Asli:
%s

Instruksi:
1. Buat Judul baru yang menarik, clickbait dikit gapapa tapi jangan bohong.
2. Tulis ulang kontennya dengan bahasa santai, singkat, gaul (pake lo-gw atau aku-kalian), dan mudah dimengerti anak sekolah.
3. GUNAKAN FORMAT HTML untuk kontennya. Gunakan tag <p>, <strong>, <em>, <ul>, <ol>, <li>, <blockquote>. Jangan gunakan Markdown.
4. Di akhir post, WAJIB kasih pertanyaan pemantik diskusi buat teman-teman.
5. Outputnya HARUS format JSON: {"title": "Judul Baru", "content": "Konten HTML Baru"}
`, title, content)

	c.model.ResponseMIMEType = "application/json"
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", "", err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", "", fmt.Errorf("no response from LLM")
	}

	// Simple parsing since we requested JSON but getting exact fields might need struct unmarshalling
	type Response struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	var result Response
	// Unmarshal the Part if it is Text
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			if err := json.Unmarshal([]byte(txt), &result); err != nil {
				// Fallback if not pure JSON? Or just return error.
				// Given setMimeType(application/json), it should be clean.
				return "", "", fmt.Errorf("failed to parse JSON: %w", err)
			}
			return result.Title, result.Content, nil
		}
	}

	return "", "", fmt.Errorf("no text content in response")
}

func (c *LLMClient) Close() {
	c.client.Close()
}
