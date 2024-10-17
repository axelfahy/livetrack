package model

import (
	"database/sql/driver"
	"fmt"
	"math"
	"time"
)

type Point struct {
	DateTime    time.Time `db:"unix_time"`
	Latitude    float64   `db:"latitude"`
	Longitude   float64   `db:"longitude"`
	Altitude    int       `db:"altitude"`
	MsgType     string    `db:"msg_type"`
	MsgContent  string    `db:"msg_content"`
	FlightTime  time.Duration
	TakeOffDist float64
	CumDist     float64
	AvgSpeed    float64
	LegSpeed    float64
	LegDist     float64
}

// Value represent a point in the database.
func (p *Point) Value() (driver.Value, error) {
	return fmt.Sprintf("(%f,%f,%d,%s,%s)", p.Latitude, p.Longitude, p.Altitude, p.MsgType, p.MsgContent), nil
}

func (p *Point) GetItineraryURL() string {
	baseURL := "https://www.google.com/maps/dir/?api=1&destination="
	linkName := "[Pick Me]"

	return fmt.Sprintf("%s(%s%f,%f&travelmode=driving)", linkName, baseURL, p.Latitude, p.Longitude)
}

// TODO: find better names
// TODO: assert that some values exist.
func (p *Point) ComputeFlightTime(previous *Point) {
	p.FlightTime = p.DateTime.Sub(previous.DateTime)
}

func (p *Point) ComputeTakeOffDist(start *Point) {
	p.TakeOffDist = distance(start.Latitude, start.Longitude, p.Latitude, p.Longitude)
}

func (p *Point) ComputeCumDist(previous *Point) {
	p.CumDist = previous.CumDist + p.LegDist
}

func (p *Point) ComputeAvgSpeed(previous *Point) {
	// TODO
}

func (p *Point) ComputeLegSpeed(previous *Point) {
	// TODO -> legDist / diff datetime previous and current point
}

func (p *Point) ComputeLegDist(previous *Point) {
	p.LegDist = distance(previous.Latitude, previous.Longitude, p.Latitude, p.Longitude)
}

func ComputeStatistics([]Point) ([]Point, error) {
	// point 1: all zero
	// iterate from point 2 to the end
	// compute stats between each points.
	// OR
	// take point 0 a base, and have diff function between points -> YES
	return nil, nil
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radlat1 := math.Pi * lat1 / 180
	radlat2 := math.Pi * lat2 / 180

	theta := lng1 - lng2
	radtheta := math.Pi * theta / 180

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515

	return dist * 1.609344
}
