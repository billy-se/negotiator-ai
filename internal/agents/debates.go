package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"golang.org/x/sync/errgroup"
)

type Engine struct {
	client openai.Client
	model  string
}

type Scenario struct {
	Scenario       string `json:"scenario"`
	ProbabilityPct int    `json:"probability_pct"`
	WatchSignal    string `json:"watch_signal"`
}

type Verdict struct {
	BullSummary string     `json:"bull_summary"`
	BearSummary string     `json:"bear_summary"`
	HypeScore   int        `json:"hype_score"`
	Scenarios   []Scenario `json:"scenarios"`
}

func NewEngine(apiKey, baseURL, model string) *Engine {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &Engine{client: client, model: model}
}

func (e *Engine) RunDebate(ctx context.Context, article, marketData string) (*Verdict, error) {
	var bullOpening, bearOpening string

	g1, gCtx1 := errgroup.WithContext(ctx)

	g1.Go(func() error {
		prompt := fmt.Sprintf("News: %s\nMarket Data: %s", article, marketData)
		res, err := e.askAgent(gCtx1,
			"You are a ruthless Bullish Market Analyst. Argue ONLY the upside catalysts and momentum opportunities. Be concise (max 3 bullet points).",
			prompt, false)
		bullOpening = res
		return err
	})

	g1.Go(func() error {
		prompt := fmt.Sprintf("News: %s\nMarket Data: %s", article, marketData)
		res, err := e.askAgent(gCtx1,
			"You are a cynical Bearish Skeptic. Attack the news for hype, missing data, or priced-in expectations using the hard market numbers. Be concise (max 3 bullet points).",
			prompt, false)
		bearOpening = res
		return err
	})

	if err := g1.Wait(); err != nil {
		return nil, fmt.Errorf("round 1 opening failed: %w", err)
	}

	var bullRebuttal, bearRebuttal string
	g2, gCtx2 := errgroup.WithContext(ctx)

	g2.Go(func() error {
		prompt := fmt.Sprintf(
			"News: %s\nMarket Data: %s\n\nYour Opening Argument:\n%s\nBear's Counter-Argument:\n%s\n\nDirectly dismantle the Bear's strongest point in 2 concise bullet points.",
			article, marketData, bullOpening, bearOpening,
		)
		res, err := e.askAgent(gCtx2,
			"You are ruthless Bullish Market Analyst in a live debate. Attack the Bear's pessimism and defend the upside thesis using hard numbers.",
			prompt, false)
		bullRebuttal = res
		return err
	})

	g2.Go(func() error {
		prompt := fmt.Sprintf(
			"News: %s\nMarket Data: %s\n\nYour Opening Argument:\n%s\n\nBull's Counter-Argument:\n%s\n\nDirectly expose the flaws in the Bull's optimism in 2 concise bullet points.",
			article, marketData, bearOpening, bullOpening,
		)
		res, err := e.askAgent(gCtx2,
			"You are a cynical Bearish Skeptic in a live debate. Expose every assumption or risk the Bull ignored using the live market data.",
			prompt, false)
		bearRebuttal = res
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, fmt.Errorf("Round 2 rebuttal failed: %w", err)
	}

	judgeInput := fmt.Sprintf("News:\n%s\n\nMarket Data:\n%s\n\n--- ROUND 1: OPENING ---\nBull:\n%s\n\nBear:\n%s\n\n--- ROUND 2: REBUTTALS ---\nBull Rebuttal:\n%s\n\nBear Rebuttal:\n%s",
		article, marketData, bullOpening, bearOpening, bullRebuttal, bearRebuttal,
	)
	judgeSystem := `You are an objective Market Negotiator judging a two-round debate. Weigh both opening statements AND how well each side survived the opponent's rebuttal against the raw market data.
	Output ONLY valid JSON matching this structure:
	{
	"bull_summary": "string",
	"bear_summary": "string",
	"hype_score": 1,
	"scenarios": [
		{"scenario": "string", "probability_pct": 50, "watch_signal": "string"}
	]
	}`

	rawJSON, err := e.askAgent(ctx, judgeSystem, judgeInput, true)
	if err != nil {
		return nil, err
	}

	var verdict Verdict
	if err := json.Unmarshal([]byte(rawJSON), &verdict); err != nil {
		return nil, fmt.Errorf("failed to parse verdict JSON: %w", err)
	}
	return &verdict, nil
}

func (e *Engine) askAgent(ctx context.Context, systemPrompt, userPrompt string, jsonMode bool) (string, error) {
	params := openai.ChatCompletionNewParams{
		Model: e.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
	}

	if jsonMode {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := e.client.Chat.Completions.New(ctx, params)
		if err == nil {
			if len(resp.Choices) == 0 {
				return "", fmt.Errorf("empty response from API")
			}
			return resp.Choices[0].Message.Content, nil
		}

		lastErr = err
		wait := time.Duration(5*(attempt+1)) * time.Second
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
	}

	return "", fmt.Errorf("agent failed after retries: %w", lastErr)
}
