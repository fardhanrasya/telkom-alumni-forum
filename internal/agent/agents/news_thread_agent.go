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

	// MaxPostsPerDay adalah plafon thread yang boleh diposting bot per hari
	// (WIB), lintas semua feed dan lintas semua run cron hari itu — bukan
	// per-feed atau per-run. Ditegakkan lewat Redis, sama pola dengan
	// MaxDailyThreadPoints di leaderboard: cegah spam kalau cron jalan lebih
	// dari sekali sehari atau ada beberapa feed yang sama-sama punya item baru.
	MaxPostsPerDay int
}

// DefaultNewsThreadConfig mengembalikan konfigurasi default
func DefaultNewsThreadConfig() NewsThreadConfig {
	return NewsThreadConfig{
		Schedule:            "0 8 * * *", // sekali sehari, jam 8 pagi WIB
		BotUsername:         "Mading_Bot",
		PreferredCategories: []string{"Teknologi", "Berita", "Umum"},
		RSSFeeds: []string{
			"https://www.cnbcindonesia.com/news/rss",
		},
		MaxItemsPerFeed:   5,
		MinContentLength:  100,
		DelayBetweenPosts: 10 * time.Second,
		RedisKeyPrefix:    "agent:news_thread",
		MaxPostsPerDay:    1,
	}
}

var newsAgentWIB = time.FixedZone("WIB", 7*3600)

// postsTodayKey is the Redis counter key for MaxPostsPerDay, scoped to the
// current WIB calendar date — same WIB-midnight rollover as daily missions,
// not UTC.
func (a *NewsThreadAgent) postsTodayKey() string {
	return fmt.Sprintf("%s:posts:%s", a.config.RedisKeyPrefix, time.Now().In(newsAgentWIB).Format("2006-01-02"))
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

	dailyCap := a.config.MaxPostsPerDay
	if dailyCap <= 0 {
		dailyCap = 1
	}

	postedToday, err := a.redis.Get(ctx, a.postsTodayKey()).Int()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to read daily post count: %w", err)
	}
	remaining := dailyCap - postedToday
	if remaining <= 0 {
		log.Printf("[%s] Daily cap (%d) already reached, skipping run", a.GetName(), dailyCap)
		return nil
	}

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

	// 3. Process RSS feeds, stopping as soon as the daily cap is hit —
	// across all feeds and all runs today, not per-feed or per-run.
	totalProcessed := 0
	for _, feedURL := range a.config.RSSFeeds {
		if remaining <= 0 {
			break
		}
		processed, err := a.processFeed(ctx, feedURL, botUser.ID, targetCategoryID, remaining)
		if err != nil {
			log.Printf("[%s] Error processing feed %s: %v", a.GetName(), feedURL, err)
			continue
		}
		totalProcessed += processed
		remaining -= processed
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

// processFeed memproses satu RSS feed, berhenti begitu budget habis
func (a *NewsThreadAgent) processFeed(ctx context.Context, feedURL string, botUserID, categoryID uuid.UUID, budget int) (int, error) {
	log.Printf("[%s] Processing feed: %s", a.GetName(), feedURL)

	// Fetch RSS items
	items, err := a.rssFetcher.FetchFeed(feedURL, a.config.MaxItemsPerFeed)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch RSS: %w", err)
	}

	processedCount := 0
	for _, item := range items {
		if processedCount >= budget {
			break
		}

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

		// Count this post against the daily cap, valid ~26h so a run just
		// before WIB midnight still expires into the next day's key.
		a.redis.Incr(ctx, a.postsTodayKey())
		a.redis.Expire(ctx, a.postsTodayKey(), 26*time.Hour)

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
