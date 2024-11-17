package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Pilot struct {
	ID          string `db:"id"`
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
		return "", fmt.Errorf("error parsing uri %s for sbb itinerary: %w", apiSearch, err)
	}

	url.RawQuery = params.Encode()

	client := &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%v", url))
	if err != nil {
		return "", fmt.Errorf("error getting sbb itinerary %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching sbb itinerary, code=%d, err=%w", resp.StatusCode, resp.Request.Context().Err())
	}

	response := struct {
		URL string `json:"url"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding sbb itinerary: %w", err)
	}

	return response.URL, nil
}

func (p *Pilot) GetTakeOffDistance() float64 {
	startPoint := p.Points[0]
	endPoint := p.Points[len(p.Points)-1]

	return distance(startPoint.Latitude, startPoint.Longitude, endPoint.Latitude, endPoint.Longitude)
}
