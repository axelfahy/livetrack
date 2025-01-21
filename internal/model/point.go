package model

import (
	"database/sql/driver"
	"fmt"
	"math"
	"time"
)

type Point struct {
	DateTime    time.Time     `json:"dateTime"    db:"unix_time"`
	Latitude    float64       `json:"latitude"    db:"latitude"`
	Longitude   float64       `json:"longitude"   db:"longitude"`
	Altitude    int           `json:"altitude"    db:"altitude"`
	MsgType     string        `json:"msgType"     db:"msg_type"`
	MsgContent  string        `json:"msgContent"  db:"msg_content"`
	FlightTime  time.Duration `json:"flightTime"`
	TakeOffDist float64       `json:"takeOffDist"`
	CumDist     float64       `json:"cumDist"`
	AvgSpeed    float64       `json:"avgSpeed"`
	LegSpeed    float64       `json:"legSpeed"`
	LegDist     float64       `json:"legDist"`
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
func (p *Point) ComputeFlightTime(start Point) {
	p.FlightTime = p.DateTime.Sub(start.DateTime)
}

func (p *Point) ComputeTakeOffDist(start Point) {
	p.TakeOffDist = distance(start.Latitude, start.Longitude, p.Latitude, p.Longitude)
}

func (p *Point) ComputeCumDist(previous Point) {
	p.CumDist = previous.CumDist + p.LegDist
}

func (p *Point) ComputeAvgSpeed() {
	p.AvgSpeed = p.CumDist / p.FlightTime.Hours()
}

func (p *Point) ComputeLegSpeed(_ Point) {
	// TODO -> legDist / diff datetime previous and current point
}

func (p *Point) ComputeLegDist(previous Point) {
	p.LegDist = distance(previous.Latitude, previous.Longitude, p.Latitude, p.Longitude)
}

func ComputeStatistics(points []Point) []Point {
	pointsWithStats := []Point{}

	for i, point := range points {
		// Skip the first point.
		if i == 0 {
			pointsWithStats = append(pointsWithStats, points[0])

			continue
		}

		previous := pointsWithStats[i-1]

		point.ComputeFlightTime(points[0])
		point.ComputeTakeOffDist(points[0])
		point.ComputeLegDist(previous)
		point.ComputeCumDist(previous)
		point.ComputeAvgSpeed()

		pointsWithStats = append(pointsWithStats, point)
	}

	return pointsWithStats
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
