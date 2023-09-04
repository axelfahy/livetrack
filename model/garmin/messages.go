package garmin

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fahy.xyz/livetrack/model"
)

type Document struct {
	XMLName    xml.Name    `xml:"kml"`
	Placemarks []Placemark `xml:"Document>Folder>Placemark"`
}

type Placemark struct {
	XMLName xml.Name `xml:"Placemark"`
	Data    []Data   `xml:"ExtendedData>Data"`
}

type Data struct {
	XMLName xml.Name `xml:"Data"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value"`
}

func (r *Document) ToPoints() ([]model.Point, error) {
	points := []model.Point{}
	for _, placemark := range r.Placemarks {
		// Last placemark does not have points, skip it.
		if len(placemark.Data) == 0 {
			continue
		}
		dateTime, err := time.Parse("1/2/2006 3:04:05 PM", placemark.getField("Time UTC"))
		if err != nil {
			return nil, err
		}
		latitude, err := strconv.ParseFloat(placemark.getField("Latitude"), 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing latitude %s: %w", placemark.getField("Latitude"), err)
		}
		longitude, err := strconv.ParseFloat(placemark.getField("Longitude"), 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing longitude %s: %w", placemark.getField("Longitude"), err)
		}
		elevation, err := strconv.Atoi(strings.Split(placemark.getField("Elevation"), ".")[0])
		if err != nil {
			return nil, fmt.Errorf("error parsing altitude %s: %w", placemark.getField("Elevation"), err)
		}
		points = append(points, model.Point{
			DateTime:   dateTime,
			Latitude:   latitude,
			Longitude:  longitude,
			Altitude:   elevation,
			MsgType:    placemark.getField("Event"),
			MsgContent: placemark.getField("Text"),
		})
	}
	return points, nil
}

func (p *Placemark) getField(field string) string {
	for _, d := range p.Data {
		if d.Name == field {
			return d.Value
		}
	}
	return ""
}

func Parse(content []byte) (Document, error) {
	var document Document
	if err := xml.Unmarshal(content, &document); err != nil {
		return Document{}, err
	}
	return document, nil
}
