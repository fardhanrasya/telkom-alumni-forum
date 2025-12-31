package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	categoryRepo "anoa.com/telkomalumiforum/internal/modules/category/repository"
	threadDto "anoa.com/telkomalumiforum/internal/modules/thread/dto"
	thread "anoa.com/telkomalumiforum/internal/modules/thread/service"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

type Agent struct {
	cron          *cron.Cron
	threadService thread.Service
	userRepo      userRepo.UserRepository
	categoryRepo  categoryRepo.CategoryRepository
	redis         *redis.Client
}

func NewAgent(threadService thread.Service, userRepo userRepo.UserRepository, categoryRepo categoryRepo.CategoryRepository, redis *redis.Client) *Agent {
	// Initialize cron with seconds precision if needed, but standard minute precision is fine.
	// Standard cron is minute-based.
	return &Agent{
		cron:          cron.New(),
		threadService: threadService,
		userRepo:      userRepo,
		categoryRepo:  categoryRepo,
		redis:         redis,
	}
}

func (a *Agent) Start() {
	// Run at 7 AM and 7 PM
	_, err := a.cron.AddFunc("0 7,19 * * *", func() {
		log.Println("🤖 Agent waking up to check news...")
		if err := a.RunJob(); err != nil {
			log.Printf("❌ Agent job failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("Failed to schedule agent job: %v", err)
	}
	a.cron.Start()
	log.Println("🤖 Agent started with schedule: 0 7,19 * * *")
}

func (a *Agent) RunJob() error {
	ctx := context.Background()

	// 1. Initialize LLM
	llm, err := NewLLMClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to init LLM: %w", err)
	}
	defer llm.Close()

	// 2. Identify Bot User
	botUser, err := a.userRepo.FindByUsername(ctx, "Mading_Bot")
	if err != nil {
		// Log and optionally create if not exists?
		// For now fail.
		return fmt.Errorf("bot user Mading_Bot not found: %w", err)
	}

	// 3. Get Category "Teknologi" or "Berita"
	categories, err := a.categoryRepo.FindAll(ctx, "")
	if err != nil {
		return err
	}
	var targetCategoryID uuid.UUID
	for _, cat := range categories {
		if cat.Name == "Teknologi" || cat.Name == "Berita" || cat.Name == "Umum" {
			targetCategoryID = cat.ID
			break
		}
	}
	if targetCategoryID == uuid.Nil && len(categories) > 0 {
		targetCategoryID = categories[0].ID
	}

	// 4. Fetch RSS
	feeds := []string{
		"https://www.cnbcindonesia.com/news/rss",
	}

	for _, url := range feeds {
		items, err := FetchRSS(url)
		if err != nil {
			log.Printf("Failed to fetch RSS %s: %v", url, err)
			continue
		}

		for _, item := range items {
			// Check if already processed
			isProcessed, err := a.redis.SIsMember(ctx, "agent:processed_urls", item.Link).Result()
			if err == nil && isProcessed {
				continue
			}

			// Scrape
			log.Printf("Processing: %s", item.Title)
			content, err := ScrapeContent(item.Link)
			if err != nil {
				log.Printf("Failed to scrape %s: %v", item.Link, err)
				continue
			}

			if len(content) < 100 {
				log.Println("Content too short, skipping")
				continue
			}

			// Rewrite
			newTitle, newContent, err := llm.RewriteNews(ctx, item.Title, content)
			if err != nil {
				log.Printf("LLM failed: %v", err)
				continue
			}

			// Add Source
			newContent += fmt.Sprintf("\n\n---\nSumber: [%s](%s)", item.Title, item.Link)

			// Post
			req := threadDto.CreateThreadRequest{
				Title:      newTitle,
				Content:    newContent,
				CategoryID: targetCategoryID.String(),
				Audience:   "semua",
			}

			if err := a.threadService.CreateThread(ctx, botUser.ID, req); err != nil {
				log.Printf("Failed to create thread: %v", err)
				continue
			}

			log.Printf("✅ Posted: %s", newTitle)

			// Mark as processed
			a.redis.SAdd(ctx, "agent:processed_urls", item.Link)

			// Wait a bit to be polite
			time.Sleep(10 * time.Second)
		}
	}

	return nil
}
