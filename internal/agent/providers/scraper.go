package providers

import (
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mmcdole/gofeed"
)

// NewsItem merepresentasikan item berita dari RSS feed
type NewsItem struct {
	Title       string
	Link        string
	Description string
	Published   string
}

// RSSFetcher mengambil berita dari RSS feed
type RSSFetcher struct {
	parser *gofeed.Parser
}

// NewRSSFetcher membuat instance baru RSS fetcher
func NewRSSFetcher() *RSSFetcher {
	return &RSSFetcher{
		parser: gofeed.NewParser(),
	}
}

// FetchFeed mengambil berita dari URL RSS feed
// limit: jumlah maksimal item yang diambil (0 = unlimited)
func (f *RSSFetcher) FetchFeed(url string, limit int) ([]NewsItem, error) {
	feed, err := f.parser.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var items []NewsItem

	// Determine actual limit
	actualLimit := len(feed.Items)
	if limit > 0 && limit < actualLimit {
		actualLimit = limit
	}

	for i := 0; i < actualLimit; i++ {
		item := feed.Items[i]
		items = append(items, NewsItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Published:   item.Published,
		})
	}

	return items, nil
}

// WebScraper mengambil konten lengkap dari halaman web
type WebScraper struct {
	collector *colly.Collector
}

// NewWebScraper membuat instance baru web scraper
func NewWebScraper() *WebScraper {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	return &WebScraper{
		collector: c,
	}
}

// ScrapeArticle mengambil konten artikel dari URL
// Menggunakan heuristik umum untuk ekstraksi konten artikel
func (s *WebScraper) ScrapeArticle(url string) (string, error) {
	var contentBuilder strings.Builder

	// Clone collector untuk request ini agar thread-safe
	c := s.collector.Clone()

	// Heuristik untuk mengambil konten artikel
	// Target elemen umum: article, .detail__body-text (Detik), .c-entry-content (TheVerge), dll
	c.OnHTML("article, .detail__body-text, .c-entry-content, #main-content, .article-content", func(e *colly.HTMLElement) {
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			text := strings.TrimSpace(el.Text)
			// Hanya ambil paragraf yang cukup panjang (filter ads/noise)
			if len(text) > 50 {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
	})

	err := c.Visit(url)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(contentBuilder.String()), nil
}

// Close membersihkan resources (optional, colly tidak butuh explicit close)
func (s *WebScraper) Close() {
	// No-op for now, colly doesn't need explicit cleanup
}
