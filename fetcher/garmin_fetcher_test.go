package fetcher

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGarminFetcher_createUrl(t *testing.T) {
	fetcherA := NewGarminFetcher(
		"https://share.garmin.com/Feed/Share/",
		slog.Default().With("component", "garmin-fetcher"),
		&emptyFetcherMetrics{},
	)
	urlA, err := fetcherA.createUrl("garminId")
	assert.Nil(t, err)
	year, month, day := time.Now().Date()
	assert.Equal(
		t,
		fmt.Sprintf("https://share.garmin.com/Feed/Share/garminId?d1=%s&d2=%s", time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04"), time.Date(year, month, day, 23, 59, 0, 0, time.UTC).Format("2006-01-02T15:04")),
		urlA,
	)
}

func TestGarminFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data, err := os.ReadFile("../model/garmin/testdata/feed.kml")
		assert.Nil(t, err)
		_, _ = w.Write(data)
	}))
	defer server.Close()
	fetcher := NewGarminFetcher(server.URL, slog.Default().With("component", "garmin-fetcher"), &emptyFetcherMetrics{})
	res, err := fetcher.Fetch("garminId")
	assert.Nil(t, err)

	assert.Equal(t, "Tracking turned on from device.", res[0].MsgType)
	assert.Equal(t, 46.62515, res[1].Latitude)
}
