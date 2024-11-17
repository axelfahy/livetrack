package fetcher

import (
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
		client:  &http.Client{Timeout: time.Duration(10) * time.Second},
		url:     url,
		logger:  logger,
		metrics: metrics,
	}
}

func (f *SpotFetcher) createUrl(id string) (string, error) {
	s, err := url.JoinPath(f.url, id, "message.json")
	if err != nil {
		return "", err
	}

	sWithDate := fmt.Sprintf("%s?startDate=%s", s, time.Now().Format("2006-01-02T00:00:00-0000"))

	return sWithDate, nil
}

func (f *SpotFetcher) Fetch(id string) ([]model.Point, error) {
	url, err := f.createUrl(id)
	if err != nil {
		return nil, err
	}

	f.logger.Info("fetching", "url", url)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response, err := spot.Parse(body)
	if err != nil {
		return nil, err
	}

	points, err := response.ToPoints()
	if err != nil {
		return nil, err
	}

	f.metrics.MessageFetched("spot")

	return points, nil
}
