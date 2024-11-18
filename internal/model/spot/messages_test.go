package spot_test

import (
	"os"
	"testing"
	"time"

	"fahy.xyz/livetrack/internal/model/spot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessagesToPoint(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/response_full.json")
	require.NoError(t, err)

	response, err := spot.Parse(content)
	require.NoError(t, err)

	points, err := response.ToPoints()
	require.NoError(t, err)
	assert.Len(t, points, 4)
	assert.Equal(t, "OK", points[0].MsgType)
	assert.Equal(t, "Pilot has landed safely", points[0].MsgContent)
	assert.Equal(t, "UNLIMITED-TRACK", points[1].MsgType)
	assert.InEpsilon(t, 46.45669, points[1].Latitude, 0.1)
	assert.InEpsilon(t, 6.88411, points[1].Longitude, 0.1)
	assert.Equal(t, time.Date(2023, time.Month(1), 14, 7, 47, 9, 0, time.UTC), points[1].DateTime)
}

func TestMessageSingleParse(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/response_single_msg.json")
	require.NoError(t, err)

	response, err := spot.Parse(content)
	require.NoError(t, err)

	assert.Equal(t, "abc", response.FeedMessageResponse.Feed.Name)
	assert.Equal(t, "abc-gen3", response.FeedMessageResponse.Messages.Message[0].MessengerName)
}

func TestMessagesParse(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/response_full.json")
	require.NoError(t, err)

	response, err := spot.Parse(content)
	require.NoError(t, err)

	assert.Equal(t, "NEW PILOT", response.FeedMessageResponse.Feed.Name)
	assert.Equal(t, "Pilot Spot", response.FeedMessageResponse.Messages.Message[0].MessengerName)
}
