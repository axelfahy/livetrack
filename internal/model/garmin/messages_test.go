package garmin_test

import (
	"os"
	"testing"
	"time"

	"fahy.xyz/livetrack/internal/model/garmin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyMessagesToPoint(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed_empty.kml")
	require.NoError(t, err)

	document, err := garmin.Parse(content)
	require.NoError(t, err)

	points, err := document.ToPoints()
	require.NoError(t, err)
	assert.Empty(t, points)
}

func TestMessagesToPoint(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed.kml")
	require.NoError(t, err)

	document, err := garmin.Parse(content)
	require.NoError(t, err)

	points, err := document.ToPoints()
	require.NoError(t, err)
	// There are 38 placemarks, but only 37 points.
	assert.Len(t, points, 37)
	assert.Equal(t, "Tracking turned on from device.", points[0].MsgType)
	assert.InEpsilon(t, 46.625150, points[1].Latitude, 0.1)
	assert.InEpsilon(t, 7.206108, points[1].Longitude, 0.1)
	assert.Equal(t, time.Date(2023, time.Month(8), 23, 10, 26, 45, 0, time.UTC), points[1].DateTime)
	assert.Equal(t, time.Date(2023, time.Month(8), 23, 10, 36, 45, 0, time.UTC), points[2].DateTime)
}

func TestMessagesParse(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed.kml")
	require.NoError(t, err)

	document, err := garmin.Parse(content)
	require.NoError(t, err)

	assert.Len(t, document.Placemarks, 38)
}

func TestMessagesParseEmpty(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed_empty.kml")
	require.NoError(t, err)

	document, err := garmin.Parse(content)
	require.NoError(t, err)

	assert.Empty(t, document.Placemarks)
}
