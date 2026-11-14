package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetCryptoSnapshot(ctx context.Context, coinIDs []string) (string, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   "api." + "coingecko.com",
		Path:   "/api/v3/simple/price",
	}
	q := u.Query()
	q.Set("ids", strings.Join(coinIDs, ","))
	q.Set("vs_currencies", "usd")
	q.Set("include_24hr_change", "true")
	q.Set("include_24hr_vol", "true")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("coingecko request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("coingecko returned status %d", resp.StatusCode)
	}

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode market data: %w", err)
	}

	var lines []string
	for _, id := range coinIDs {
		metrics, ok := data[id]
		if !ok {
			continue
		}
		line := fmt.Sprintf(
			"Asset: %s | Price: $%.2f | 24h Change: %+.2f%% | 24h Volume: $%.0f",
			strings.ToUpper(id),
			metrics["usd"],
			metrics["usd_24h_change"],
			metrics["usd_24h_vol"],
		)
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return "No live market data available.", nil
	}
	return strings.Join(lines, "\n"), nil
}
