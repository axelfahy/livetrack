package model_test

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"

	"fahy.xyz/livetrack/internal/model"
	"fahy.xyz/livetrack/internal/model/spot"
	"github.com/stretchr/testify/assert"
)

const trackFile = "spot/testdata/response_long_full.json"

var (
	logger = slog.New(slog.Default().Handler())
	pilot  model.Pilot
)

func TestMain(m *testing.M) {
	content, err := os.ReadFile(trackFile)
	if err != nil {
		logger.Error("Could not read file", "file", trackFile, "error", err)
		os.Exit(1)
	}

	response, err := spot.Parse(content)
	if err != nil {
		logger.Error("Could not parse content", "content", content, "error", err)
		os.Exit(1)
	}

	points, err := response.ToPoints()
	if err != nil {
		logger.Error("Could not convert to points", "error", err)
		os.Exit(1)
	}

	// Reverse the points because the DB Manager returns the points ascending.
	slices.Reverse(points)

	pilot = model.Pilot{
		Name:   "test",
		Points: points,
		Home:   "test-home",
	}

	code := m.Run()
	os.Exit(code)
}

func TestPilot_GetCumulativeDistance(t *testing.T) {
	t.Parallel()

	dist := pilot.GetCumulativeDistance()
	assert.InEpsilon(t, 143.90572292269138, dist, 0.1)
}

func TestPilot_GetFlightTime(t *testing.T) {
	t.Parallel()

	flightTime := pilot.GetFlightTime()
	assert.Equal(t, "4h12m11s", fmt.Sprint(flightTime))
}

func TestPilot_GetLivetrackURL(t *testing.T) {
	t.Parallel()

	url := pilot.GetLivetrackURL("https://test.xyz/")
	assert.Equal(t, "[Livetrack](https://test.xyz/?pilot=test)", url)
}

func TestPilot_GetTakeOffDistance(t *testing.T) {
	t.Parallel()

	dist := pilot.GetTakeOffDistance()
	assert.InEpsilon(t, 129.94354857890977, dist, 0.1)
}
