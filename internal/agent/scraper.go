package agent

import (
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mmcdole/gofeed"
)

type NewsItem struct {
	Title string
	Link  string
	Body  string
}

func FetchRSS(url string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var items []NewsItem
	// Limit to top 5 to avoid spamming
	limit := 5
	if len(feed.Items) < limit {
		limit = len(feed.Items)
	}

	for i := 0; i < limit; i++ {
		item := feed.Items[i]
		items = append(items, NewsItem{
			Title: item.Title,
			Link:  item.Link,
		})
	}
	return items, nil
}

func ScrapeContent(url string) (string, error) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	
	// Default text
	var contentBuilder strings.Builder

	// Heuristik untuk mengambil konten artikel (umumnya dalam <p> di dalam article atau div main)
	// Kita coba target umum dulu. Detik biasanya di .detail__body-text
	// TheVerge di .c-entry-content
	// Kita ambil semua <p> yang panjangnya lumayan.
	
	c.OnHTML("article, .detail__body-text, .c-entry-content, #main-content", func(e *colly.HTMLElement) {
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			text := strings.TrimSpace(el.Text)
			if len(text) > 50 { // Hanya ambil paragraf yang cukup panjang
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
	})

	err := c.Visit(url)
	if err != nil {
		return "", err
	}

	return contentBuilder.String(), nil
}
