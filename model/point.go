package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Point struct {
	DateTime    time.Time `db:"unix_time"`
	Latitude    float64   `db:"latitude"`
	Longitude   float64   `db:"longitude"`
	Altitude    int       `db:"altitude"`
	MsgType     string    `db:"msg_type"`
	MsgContent  string    `db:"msg_content"`
	FlightTime  int
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
