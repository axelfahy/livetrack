package fetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"fahy.xyz/livetrack/internal/model"
	"fahy.xyz/livetrack/internal/model/spot"
)

type SpotFetcher struct {
	client  *http.Client
	url     string
	logger  *slog.Logger
	metrics metrics
}

func NewSpotFetcher(url string, logger *slog.Logger, metrics metrics) *SpotFetcher {
	return &SpotFetcher{
		client:  &http.Client{Timeout: HTTPTimeout},
		url:     url,
		logger:  logger,
		metrics: metrics,
	}
}

func (f *SpotFetcher) createURL(id string) (string, error) {
	s, err := url.JoinPath(f.url, id, "message.json")
	if err != nil {
		return "", fmt.Errorf("joining path: %w", err)
	}

	sWithDate := fmt.Sprintf("%s?startDate=%s", s, time.Now().Format("2006-01-02T00:00:00-0000"))

	return sWithDate, nil
}

func (f *SpotFetcher) Fetch(ctx context.Context, id string) ([]model.Point, error) {
	url, err := f.createURL(id)
	if err != nil {
		return nil, fmt.Errorf("creating URL: %w", err)
	}

	f.logger.Info("fetching", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Wget/1.13.4 (linux-gnu)")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	response, err := spot.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parsing body: %w", err)
	}

	points, err := response.ToPoints()
	if err != nil {
		return nil, fmt.Errorf("parsing points: %w", err)
	}

	f.metrics.MessageFetched("spot")

	return points, nil
}
