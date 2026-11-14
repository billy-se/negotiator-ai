# negotiator-ai

> An automated market intelligence engine built in Go where adversarial AI agents ingest breaking news, verify live market data, and negotiate probable market scenarios.

## Overview

Most AI news tools rely on single-prompt summaries that blindly echo clickbait headlines or produce overly neutral "fence-sitting" answers. **`negotiator-ai`** eliminates manual browsing and emotional bias by turning market analysis into a structured, concurrent debate.

When breaking news hits, a Go pipeline pairs the article with real-time market metrics and spins up isolated AI agents with opposing mandates. After arguing the upside and downside cases against hard numbers, a final **Negotiator Agent** synthesizes the arguments into a machine-readable JSON verdict.

## Architecture & Pipeline

1. **Ingestion Worker (`internal/ingestor`)**
   - Polls configured RSS feeds and strips embedded HTML tags into clean text.
   - Deduplicates seen URLs using SQLite to prevent redundant API calls.
2. **Market Context Enrichment (`internal/market`)**
   - Queries live CoinGecko market data (price action, 24h change, and 24h volume) at the exact moment news breaks.
   - Injects hard numerical context alongside the text so agents cannot hallucinate market reactions.
3. **Concurrent Adversarial Debate & Verdict (`internal/agents`)**
   - Uses Go routines and `errgroup` to execute opposing agents simultaneously:
     - **Bull Agent:** Argues strictly for upside catalysts, adoption signals, and momentum.
     - **Bear Agent:** Audits the claim for hype, missing data, and priced-in expectations.
   - A **Judge/Negotiator Agent** cross-examines both opening arguments against the raw market data and enforces a strict JSON schema.

## Tech Stack

- **Language:** Go 1.23+ (Goroutines & `errgroup` for concurrent agent execution)
- **LLM Orchestration:** Official OpenAI Go SDK (`openai-go`) configured for Google Gemini Flash (`gemini-3-flash-preview`)
- **Data Ingestion:** `gofeed` (RSS parsing with custom User-Agent)
- **Market Data:** CoinGecko public REST API
- **Storage:** Pure-Go SQLite (`glebarez/go-sqlite`) for article deduplication & verdict history

## Project Structure

```text
negotiator-ai/
├── cmd/
│   └── server/
│       └── main.go              # Entry point & pipeline orchestrator
├── internal/
│   ├── agents/
│   │   └── debates.go           # Concurrent Bull/Bear runners & Judge verdict parser
│   ├── ingestor/
│   │   └── rss.go               # RSS feed poller & HTML cleaner
│   ├── market/
│   │   └── client.go            # Live CoinGecko price & volume fetcher
│   └── store/
│       └── sqlite.go            # Deduplication & verdict storage
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

## Example Output Schema

Instead of a wall of conversational text, **`negotiator-ai`** produces structured, queryable verdicts:

```json
{
  "bull_summary": "The confirmation that the outage was routine maintenance removes a systemic risk factor, framing the dip as a liquidity flush.",
  "bear_summary": "Market leaders continue trading in the red on high volume, suggesting broader de-risking rather than a single news event.",
  "hype_score": 4,
  "scenarios": [
    {
      "scenario": "Market Snapback",
      "probability_pct": 45,
      "watch_signal": "Bitcoin reclaiming the \$68,500 level on high volume."
    },
    {
      "scenario": "Extended Tech De-risking",
      "probability_pct": 40,
      "watch_signal": "Solana dropping below the \$70.00 psychological support level."
    },
    {
      "scenario": "Low Volatility Consolidation",
      "probability_pct": 15,
      "watch_signal": "24h volume dropping below \$20B for BTC as market indecision takes over."
    }
  ]
}
```

## Getting Started

### Prerequisites
- Go 1.23+
- Free API key from Google AI Studio

### Installation

```bash
# Clone the repository (replace with your GitHub repo URL)
git clone https://github.com
cd negotiator-ai

# Create your .env file and add your API key
echo "LLM_API_KEY=your_api_key_here" > .env

# Install dependencies and run
go mod tidy
go run ./cmd/server
```

## Roadmap

- [x] **Phase 1:** Core RSS poller, SQLite deduplication, and concurrent Bull/Bear/Judge pipeline
- [x] **Phase 2:** Live market API integration (CoinGecko)
- [ ] **Phase 3:** Multi-round rebuttal loop (agents directly critique each other's opening statements)
- [ ] **Phase 4:** Scheduled background worker & Discord/Telegram webhook alerts
- [ ] **Phase 5:** Historical accuracy tracker (automatically grading past scenarios against actual 24h price outcomes)

Disclaimer: This project is for educational and research purposes only and does not constitute financial or investment advice.
