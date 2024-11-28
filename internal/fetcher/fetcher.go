package fetcher

import (
	"time"

	"fahy.xyz/livetrack/internal/model"
)

type Fetcher interface {
	Fetch(id string) ([]model.Point, error)
}

type metrics interface {
	MessageFetched(source string)
}

type emptyMetrics struct{}

func (m emptyMetrics) MessageFetched(string) {}

const (
	HTTPTimeout = 10 * time.Second
)
