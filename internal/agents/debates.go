package agents

import (
	"context"
	"encoding/json"
	"fmt"

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
	var bullThesis, bearThesis string

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		prompt := fmt.Sprintf("News: %s\nMarket Data: %s", article, marketData)
		res, err := e.askAgent(gCtx,
			"You are a ruthless Bullish Market Analyst. Argue ONLY the upside catalysts and momentum opportunities. Be concise (max 3 bullet points).",
			prompt, false)
		bullThesis = res
		return err
	})

	g.Go(func() error {
		prompt := fmt.Sprintf("News: %s\nMarket Data: %s", article, marketData)
		res, err := e.askAgent(gCtx,
			"You are a cynical Bearish Skeptic. Attack the news for hype, missing data, or priced-in expectations using the hard market numbers. Be concise (max 3 bullet points).",
			prompt, false)
		bearThesis = res
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("debate failed: %w", err)
	}

	judgeInput := fmt.Sprintf("News:\n%s\n\nMarket Data:\n%s\n\nBull Argument:\n%s\n\nBear Argument:\n%s",
		article, marketData, bullThesis, bearThesis)
	judgeSystem := `You are an objective Market Negotiator. Weigh the Bull and Bear arguments against the raw market data.
	Output ONLY valid JSON matching this structure:
	{
		"bull_summary": "string",
		"bear_summary": "string",
		"hype_score": 1-10,
		"scenarios": [
			{"scenario": "string", "probability_pct": number, "watch_signal": "string"}
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

	resp, err := e.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty responses from API")
	}
	return resp.Choices[0].Message.Content, nil
}
