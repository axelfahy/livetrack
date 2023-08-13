package fetcher

import "fahy.xyz/livetrack/model"

type Fetcher interface {
	Fetch(id string) ([]model.Point, error)
}

type fetcherMetrics interface {
	MessageFetchedInc(source string)
}

type emptyFetcherMetrics struct{}

func (m emptyFetcherMetrics) MessageFetchedInc(source string) {}
