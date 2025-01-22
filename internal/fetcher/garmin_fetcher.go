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
	"fahy.xyz/livetrack/internal/model/garmin"
)

type GarminFetcher struct {
	client  *http.Client
	url     string
	logger  *slog.Logger
	metrics metrics
}

func NewGarminFetcher(url string, logger *slog.Logger, metrics metrics) *GarminFetcher {
	return &GarminFetcher{
		client:  &http.Client{Timeout: HTTPTimeout},
		url:     url,
		logger:  logger,
		metrics: metrics,
	}
}

func (f *GarminFetcher) createURL(id string) (string, error) {
	urlWithID, err := url.JoinPath(f.url, id)
	if err != nil {
		return "", fmt.Errorf("joining path: %w", err)
	}

	year, month, day := time.Now().Date()
	sWithDate := fmt.Sprintf(
		"%s?d1=%s&d2=%s",
		urlWithID,
		time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04"),
		time.Date(year, month, day, 23, 59, 0, 0, time.UTC).Format("2006-01-02T15:04"),
	)

	return sWithDate, nil
}

func (f *GarminFetcher) Fetch(ctx context.Context, id string) ([]model.Point, error) {
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

	response, err := garmin.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parsing body: %w", err)
	}

	points, err := response.ToPoints()
	if err != nil {
		return nil, fmt.Errorf("parsing points: %w", err)
	}

	f.metrics.MessageFetched("garmin")

	return points, nil
}
