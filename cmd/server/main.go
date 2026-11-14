package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"negotiator-ai/internal/agents"
	"negotiator-ai/internal/ingestor"
	"negotiator-ai/internal/market"
	"negotiator-ai/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set LLM_API_KEY environment variable")
	}

	ctx := context.Background()

	db, err := store.NewStore("negotiator.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	engine := agents.NewEngine(
		apiKey,
		"https://generativelanguage"+".googleapis.com/v1beta/openai",
		"gemini-3-flash-preview",
	)
	rss := ingestor.NewRSSIngestor()
	mkt := market.NewClient()

	marketSnapshot, err := mkt.GetCryptoSnapshot(ctx, []string{"bitcoin", "ethereum", "solana"})
	if err != nil {
		log.Printf("Warning: market fetch failed (%v), continuing with news only", err)
		marketSnapshot = "Market data temporarily unavailable."
	}
	fmt.Println(marketSnapshot)

	articles, err := rss.FetchLatest(ctx, "https://cointelegraph"+".com/rss", 3)
	if err != nil {
		log.Fatal(err)
	}

	for _, art := range articles {
		seen, err := db.HasProcessed(art.Link)
		if err != nil {
			log.Printf("DB lookup error: %v", err)
			continue
		}
		if seen {
			fmt.Printf("\n[SKIPPED - Already Analyzed] %s\n", art.Title)
			continue
		}

		fmt.Printf("\n=== DEBATING: %s ====\n", art.Title)
		articleInput := fmt.Sprintf("Title: %s\nSummary: %s", art.Title, art.Summary)

		verdict, err := engine.RunDebate(ctx, articleInput, marketSnapshot)
		if err != nil {
			log.Printf("Debate error for %s: %v", art.Title, err)
			continue
		}

		prettyJSON, _ := json.MarshalIndent(verdict, "", "  ")
		fmt.Println(string(prettyJSON))

		if err := db.SaveVerdict(art.Link, art.Title, verdict.HypeScore, string(prettyJSON)); err != nil {
			log.Printf("Failed to save verdict: %v", err)
		}
	}
}
