package ingestor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Article struct {
	Title     string
	Link      string
	Summary   string
	Published string
}

type RSSIngestor struct {
	parser *gofeed.Parser
}

func NewRSSIngestor() *RSSIngestor {
	p := gofeed.NewParser()
	p.UserAgent = "Mozilla/5.0 (compatible; NegotiatorAI/1.0)"
	return &RSSIngestor{parser: p}
}

func (r *RSSIngestor) FetchLatest(ctx context.Context, feedURL string, limit int) ([]Article, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	feed, err := r.parser.ParseURLWithContext(feedURL, reqCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed %s: %w", feedURL, err)
	}

	var articles []Article
	for i, item := range feed.Items {
		if i >= limit {
			break
		}

		rawText := item.Description
		if rawText == "" {
			rawText = item.Content
		}

		articles = append(articles, Article{
			Title:     strings.TrimSpace(item.Title),
			Link:      strings.TrimSpace(item.Link),
			Summary:   cleanHTML(rawText),
			Published: item.Published,
		})
	}
	return articles, nil
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func cleanHTML(input string) string {
	stripped := htmlTagRegex.ReplaceAllString(input, " ")
	return strings.Join(strings.Fields(stripped), " ")
}
