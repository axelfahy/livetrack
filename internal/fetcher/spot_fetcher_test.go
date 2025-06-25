package fetcher

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpotFetcher_createURL(t *testing.T) {
	t.Parallel()

	fetcherA := NewSpotFetcher(
		"https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/",
		slog.Default().With("component", "spot-fetcher"),
		&emptyMetrics{},
	)
	urlA, err := fetcherA.createURL("0onlLopfoM4bG5jXvWRE8H0Obd0oMxMBq")
	require.NoError(t, err)
	assert.Equal(
		t,
		"https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/0onlLopfoM4bG5jXvWRE8H0Obd0oMxMBq/message.json?startDate="+time.Now().Format("2006-01-02T00:00:00-0000"),
		urlA,
	)
}

func TestSpotFetcher_Fetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		data, err := os.ReadFile("../model/spot/testdata/response_full.json")
		assert.NoError(t, err)

		_, _ = w.Write(data)
	}))
	defer server.Close()

	fetcher := NewSpotFetcher(server.URL, slog.Default().With("component", "spot-fetcher"), &emptyMetrics{})
	res, err := fetcher.Fetch(t.Context(), "0smxuLcDXXlQkR6Uzu2HcDvp7MmW7TCLc")
	require.NoError(t, err)

	assert.Equal(t, "OK", res[0].MsgType)
	assert.InEpsilon(t, 46.45669, res[1].Latitude, 0.1)
}
