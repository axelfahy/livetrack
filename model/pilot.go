package model

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"
)

type Pilot struct {
	Id          string `db:"id"`
	Name        string `db:"name"`
	Points      []Point
	Home        string   `db:"home"`
	Orgs        []string `db:"orgs"`
	TrackerType string   `db:"tracker_type"`
}

const apiSearch string = "https://timetable.search.ch/api/route.json"

func (p *Pilot) GetCumulativeDistance() float64 {
	dist := 0.0
	for i := 0; i < len(p.Points)-2; i++ {
		startPoint := p.Points[i]
		endPoint := p.Points[i+1]
		dist += distance(startPoint.Latitude, startPoint.Longitude, endPoint.Latitude, endPoint.Longitude)
	}
	return dist
}

// GetFlightTime returns the duration between the first and last point.
func (p *Pilot) GetFlightTime() time.Duration {
	return p.Points[len(p.Points)-1].DateTime.Sub(p.Points[0].DateTime)
}

// GetSbbItinerary retrieves the SBB itinerary of the pilot to go home.
func (p *Pilot) GetSbbItinerary(latitude, longitude float64) (string, error) {
	params := url.Values{}
	params.Add("from", fmt.Sprintf("%f,%f", latitude, longitude))
	params.Add("to", p.Home)

	url, err := url.ParseRequestURI(apiSearch)
	if err != nil {
		return "", err
	}
	url.RawQuery = params.Encode()

	client := &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("%v", url))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching sbb itinerary, code=%d", resp.StatusCode)
	}
	response := struct {
		Url string `string:"url"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	return response.Url, nil
}

func (p *Pilot) GetTakeOffDistance() float64 {
	startPoint := p.Points[0]
	endPoint := p.Points[len(p.Points)-1]
	return distance(startPoint.Latitude, startPoint.Longitude, endPoint.Latitude, endPoint.Longitude)
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radlat1 := float64(math.Pi * lat1 / 180)
	radlat2 := float64(math.Pi * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(math.Pi * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515

	return dist * 1.609344
}
