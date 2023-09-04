package garmin_test

import (
	"os"
	"testing"
	"time"

	"fahy.xyz/livetrack/model/garmin"
	"github.com/stretchr/testify/assert"
)

func TestEmptyMessagesToPoint(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed_empty.kml")
	assert.Nil(t, err)

	document, err := garmin.Parse(content)
	assert.Nil(t, err)

	points, err := document.ToPoints()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(points))
}

func TestMessagesToPoint(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed.kml")
	assert.Nil(t, err)

	document, err := garmin.Parse(content)
	assert.Nil(t, err)

	points, err := document.ToPoints()
	assert.Nil(t, err)
	// There are 38 placemarks, but only 37 points.
	assert.Equal(t, 37, len(points))
	assert.Equal(t, "Tracking turned on from device.", points[0].MsgType)
	assert.Equal(t, 46.625150, points[1].Latitude)
	assert.Equal(t, 7.206108, points[1].Longitude)
	assert.Equal(t, time.Date(2023, time.Month(8), 23, 10, 26, 45, 0, time.UTC), points[1].DateTime)
}

func TestMessagesParse(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed.kml")
	assert.Nil(t, err)

	document, err := garmin.Parse(content)
	assert.Nil(t, err)

	assert.Equal(t, 38, len(document.Placemarks))
}

func TestMessagesParseEmpty(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("testdata/feed_empty.kml")
	assert.Nil(t, err)

	document, err := garmin.Parse(content)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(document.Placemarks))
}
