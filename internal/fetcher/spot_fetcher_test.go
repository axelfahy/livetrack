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

func TestSpotFetcher_createUrl(t *testing.T) {
	fetcherA := NewSpotFetcher(
		"https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/",
		slog.Default().With("component", "spot-fetcher"),
		&emptyMetrics{},
	)
	urlA, err := fetcherA.createUrl("0onlLopfoM4bG5jXvWRE8H0Obd0oMxMBq")
	assert.Nil(t, err)
	assert.Equal(
		t,
		fmt.Sprintf("https://api.findmespot.com/spot-main-web/consumer/rest-api/2.0/public/feed/0onlLopfoM4bG5jXvWRE8H0Obd0oMxMBq/message.json?startDate=%s", time.Now().Format("2006-01-02T00:00:00-0000")),
		urlA,
	)
}

func TestSpotFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		data, err := os.ReadFile("../model/spot/testdata/response_full.json")
		assert.Nil(t, err)

		_, _ = w.Write(data)
	}))
	defer server.Close()
	fetcher := NewSpotFetcher(server.URL, slog.Default().With("component", "spot-fetcher"), &emptyMetrics{})
	res, err := fetcher.Fetch("0smxuLcDXXlQkR6Uzu2HcDvp7MmW7TCLc")
	assert.Nil(t, err)

	assert.Equal(t, "OK", res[0].MsgType)
	assert.Equal(t, 46.45669, res[1].Latitude)
}
