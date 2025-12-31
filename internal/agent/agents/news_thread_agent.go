package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"anoa.com/telkomalumiforum/internal/agent/providers"
	categoryRepo "anoa.com/telkomalumiforum/internal/modules/category/repository"
	threadDto "anoa.com/telkomalumiforum/internal/modules/thread/dto"
	thread "anoa.com/telkomalumiforum/internal/modules/thread/service"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// NewsThreadAgent adalah agent yang secara otomatis membuat thread dari berita RSS
type NewsThreadAgent struct {
	// Dependencies
	threadService thread.Service
	userRepo      userRepo.UserRepository
	categoryRepo  categoryRepo.CategoryRepository
	redis         *redis.Client

	// Providers
	llmProvider providers.LLMProvider
	rssFetcher  *providers.RSSFetcher
	webScraper  *providers.WebScraper

	// Configuration
	config NewsThreadConfig
}

// NewsThreadConfig adalah konfigurasi untuk NewsThreadAgent
type NewsThreadConfig struct {
	// Schedule cron (misal: "0 7,19 * * *" untuk jam 7 pagi dan 7 malam)
	Schedule string

	// BotUsername adalah username bot yang akan posting thread
	BotUsername string

	// PreferredCategories adalah daftar kategori yang diprioritaskan (optional)
	PreferredCategories []string

	// RSSFeeds adalah daftar URL RSS feed yang akan dimonitor
	RSSFeeds []string

	// MaxItemsPerFeed adalah jumlah maksimal item yang diambil per feed
	MaxItemsPerFeed int

	// MinContentLength adalah panjang minimal konten artikel (filter spam)
	MinContentLength int

	// DelayBetweenPosts adalah delay antar posting (untuk rate limiting)
	DelayBetweenPosts time.Duration

	// RedisKeyPrefix adalah prefix untuk Redis key tracking
	RedisKeyPrefix string
}

// DefaultNewsThreadConfig mengembalikan konfigurasi default
func DefaultNewsThreadConfig() NewsThreadConfig {
	return NewsThreadConfig{
		Schedule:            "0 7,19 * * *", // 7 AM & 7 PM
		BotUsername:         "Mading_Bot",
		PreferredCategories: []string{"Teknologi", "Berita", "Umum"},
		RSSFeeds: []string{
			"https://www.cnbcindonesia.com/news/rss",
		},
		MaxItemsPerFeed:   5,
		MinContentLength:  100,
		DelayBetweenPosts: 10 * time.Second,
		RedisKeyPrefix:    "agent:news_thread",
	}
}

// NewNewsThreadAgent membuat instance NewsThreadAgent baru
func NewNewsThreadAgent(
	threadService thread.Service,
	userRepo userRepo.UserRepository,
	categoryRepo categoryRepo.CategoryRepository,
	redis *redis.Client,
	llmProvider providers.LLMProvider,
	config NewsThreadConfig,
) *NewsThreadAgent {
	return &NewsThreadAgent{
		threadService: threadService,
		userRepo:      userRepo,
		categoryRepo:  categoryRepo,
		redis:         redis,
		llmProvider:   llmProvider,
		rssFetcher:    providers.NewRSSFetcher(),
		webScraper:    providers.NewWebScraper(),
		config:        config,
	}
}

// GetName implements agent.Agent
func (a *NewsThreadAgent) GetName() string {
	return "NewsThreadAgent"
}

// GetSchedule implements agent.Agent
func (a *NewsThreadAgent) GetSchedule() string {
	return a.config.Schedule
}

// Execute implements agent.Agent
func (a *NewsThreadAgent) Execute(ctx context.Context) error {
	log.Printf("[%s] Starting execution...", a.GetName())

	// 1. Get bot user
	botUser, err := a.userRepo.FindByUsername(ctx, a.config.BotUsername)
	if err != nil {
		return fmt.Errorf("bot user %s not found: %w", a.config.BotUsername, err)
	}

	// 2. Get target category
	targetCategoryID, err := a.getTargetCategory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get target category: %w", err)
	}

	// 3. Process RSS feeds
	totalProcessed := 0
	for _, feedURL := range a.config.RSSFeeds {
		processed, err := a.processFeed(ctx, feedURL, botUser.ID, targetCategoryID)
		if err != nil {
			log.Printf("[%s] Error processing feed %s: %v", a.GetName(), feedURL, err)
			continue
		}
		totalProcessed += processed
	}

	log.Printf("[%s] Execution completed. Total threads created: %d", a.GetName(), totalProcessed)
	return nil
}

// getTargetCategory mencari kategori yang sesuai untuk posting
func (a *NewsThreadAgent) getTargetCategory(ctx context.Context) (uuid.UUID, error) {
	categories, err := a.categoryRepo.FindAll(ctx, "")
	if err != nil {
		return uuid.Nil, err
	}

	// Cari kategori yang preferred
	for _, cat := range categories {
		for _, preferred := range a.config.PreferredCategories {
			if cat.Name == preferred {
				return cat.ID, nil
			}
		}
	}

	// Fallback ke kategori pertama jika tidak ada yang match
	if len(categories) > 0 {
		return categories[0].ID, nil
	}

	return uuid.Nil, fmt.Errorf("no categories available")
}

// processFeed memproses satu RSS feed
func (a *NewsThreadAgent) processFeed(ctx context.Context, feedURL string, botUserID, categoryID uuid.UUID) (int, error) {
	log.Printf("[%s] Processing feed: %s", a.GetName(), feedURL)

	// Fetch RSS items
	items, err := a.rssFetcher.FetchFeed(feedURL, a.config.MaxItemsPerFeed)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch RSS: %w", err)
	}

	processedCount := 0
	for _, item := range items {
		// Check if already processed
		redisKey := fmt.Sprintf("%s:processed_urls", a.config.RedisKeyPrefix)
		isProcessed, err := a.redis.SIsMember(ctx, redisKey, item.Link).Result()
		if err == nil && isProcessed {
			continue
		}

		// Process this item
		if err := a.processNewsItem(ctx, item, botUserID, categoryID, redisKey); err != nil {
			log.Printf("[%s] Failed to process item '%s': %v", a.GetName(), item.Title, err)
			continue
		}

		processedCount++

		// Rate limiting
		time.Sleep(a.config.DelayBetweenPosts)
	}

	return processedCount, nil
}

// processNewsItem memproses satu item berita dan membuat thread
func (a *NewsThreadAgent) processNewsItem(ctx context.Context, item providers.NewsItem, botUserID, categoryID uuid.UUID, redisKey string) error {
	log.Printf("[%s] Processing: %s", a.GetName(), item.Title)

	// 1. Scrape full content
	content, err := a.webScraper.ScrapeArticle(item.Link)
	if err != nil {
		return fmt.Errorf("failed to scrape article: %w", err)
	}

	if len(content) < a.config.MinContentLength {
		return fmt.Errorf("content too short (%d chars), skipping", len(content))
	}

	// 2. Rewrite dengan LLM
	newTitle, newContent, err := a.rewriteNewsWithLLM(ctx, item.Title, content)
	if err != nil {
		return fmt.Errorf("LLM rewrite failed: %w", err)
	}

	// 3. Add source attribution
	newContent += fmt.Sprintf("\n\n---\nSumber: [%s](%s)", item.Title, item.Link)

	// 4. Create thread
	req := threadDto.CreateThreadRequest{
		Title:      newTitle,
		Content:    newContent,
		CategoryID: categoryID.String(),
		Audience:   "semua",
	}

	if err := a.threadService.CreateThread(ctx, botUserID, req); err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}

	// 5. Mark as processed
	a.redis.SAdd(ctx, redisKey, item.Link)

	log.Printf("[%s] ✅ Posted: %s", a.GetName(), newTitle)
	return nil
}

// rewriteNewsWithLLM menggunakan LLM untuk rewrite berita
func (a *NewsThreadAgent) rewriteNewsWithLLM(ctx context.Context, title, content string) (string, string, error) {
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
3. GUNAKAN FORMAT HTML untuk kontennya (judul tidak termasuk). Gunakan tag <p>, <strong>, <em>, <ul>, <ol>, <li>, <blockquote>. Jangan gunakan Markdown.
4. Di akhir post, WAJIB kasih pertanyaan pemantik diskusi buat teman-teman.
5. Outputnya HARUS format JSON: {"title": "Judul Baru", "content": "Konten HTML Baru"}
`, title, content)

	type Response struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	var result Response
	if err := a.llmProvider.GenerateStructured(ctx, prompt, &result); err != nil {
		return "", "", err
	}

	return result.Title, result.Content, nil
}
