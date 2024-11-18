package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGarminFetcher_createURL(t *testing.T) {
	t.Parallel()

	fetcherA := NewGarminFetcher(
		"https://share.garmin.com/Feed/Share/",
		slog.Default().With("component", "garmin-fetcher"),
		&emptyMetrics{},
	)
	urlA, err := fetcherA.createURL("garminId")
	require.NoError(t, err)

	year, month, day := time.Now().Date()
	assert.Equal(
		t,
		fmt.Sprintf("https://share.garmin.com/Feed/Share/garminId?d1=%s&d2=%s", time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04"), time.Date(year, month, day, 23, 59, 0, 0, time.UTC).Format("2006-01-02T15:04")),
		urlA,
	)
}

func TestGarminFetcher_Fetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		data, err := os.ReadFile("../model/garmin/testdata/feed.kml")
		assert.NoError(t, err)

		_, _ = w.Write(data)
	}))
	defer server.Close()
	fetcher := NewGarminFetcher(server.URL, slog.Default().With("component", "garmin-fetcher"), &emptyMetrics{})
	res, err := fetcher.Fetch(context.Background(), "garminId")
	require.NoError(t, err)

	assert.Equal(t, "Tracking turned on from device.", res[0].MsgType)
	assert.InEpsilon(t, 46.62515, res[1].Latitude, 0.1)
}
