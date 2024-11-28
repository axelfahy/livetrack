package spot

import (
	"encoding/json"
	"fmt"
	"time"

	"fahy.xyz/livetrack/internal/model"
)

type Root struct {
	Response Response `json:"response"`
}

type Response struct {
	FeedMessageResponse FeedMessageResponse `json:"feedMessageResponse"`
}

type FeedMessageResponse struct {
	Count         int      `json:"count"`
	Feed          Feed     `json:"feed"`
	TotalCount    int      `json:"totalCount"`
	ActivityCount int      `json:"activityCount"`
	Messages      Messages `json:"messages"`
}

type Feed struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Status               string `json:"status"`
	Usage                int    `json:"usage"`
	DaysRange            int    `json:"daysRange"`
	DetailedMessageShown bool   `json:"detailedMessageShown"`
	Type                 string `json:"type"`
}

type Messages struct {
	Message Content `json:"message"`
}

type Content []MessageContent

// UnmarshalJSON is a custom decoder to decode message content.
//
// This is needed because if there is a single point, the content is
// not directly in a list.
func (v *Content) UnmarshalJSON(point []byte) error {
	if point[0] == '[' { // First char is '[', so it's a JSON array
		s := make([]MessageContent, 0)
		if err := json.Unmarshal(point, &s); err != nil {
			return fmt.Errorf("error unmarshalling array of point %s: %w", string(point), err)
		}

		*v = Content(s)

		return nil
	}
	// else it's a simple string
	*v = make(Content, 1)

	if err := json.Unmarshal(point, &(*v)[0]); err != nil {
		return fmt.Errorf("error unmarshalling single point %s: %w", string(point), err)
	}

	return nil
}

type MessageContent struct {
	ClientUnixTime string  `json:"@clientUnixTime"`
	ID             int     `json:"id"`
	MessengerID    string  `json:"messengerId"`
	MessengerName  string  `json:"messengerName"`
	UnixTime       int     `json:"unixType"`
	MessageType    string  `json:"messageType"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	ModelID        string  `json:"modelId"`
	ShowCustomMsg  string  `json:"showCustomMsg"`
	DateTime       string  `json:"dateTime"`
	BatteryState   string  `json:"batteryState"`
	Hidden         int     `json:"hidden"`
	MessageContent string  `json:"messageContent,omitempty"`
	Altitude       int     `json:"altitude"`
}

func (r *Response) ToPoints() ([]model.Point, error) {
	points := []model.Point{}

	for _, message := range r.FeedMessageResponse.Messages.Message {
		dateTime, err := time.Parse("2006-01-02T15:04:05+0000", message.DateTime)
		if err != nil {
			return nil, fmt.Errorf("error parsing date %s: %w", message.DateTime, err)
		}

		points = append(points, model.Point{
			DateTime:    dateTime,
			Latitude:    message.Latitude,
			Longitude:   message.Longitude,
			Altitude:    message.Altitude,
			MsgType:     message.MessageType,
			MsgContent:  message.MessageContent,
			FlightTime:  0,
			TakeOffDist: 0.0,
			CumDist:     0.0,
			AvgSpeed:    0.0,
			LegSpeed:    0.0,
			LegDist:     0.0,
		})
	}

	return points, nil
}

func Parse(content []byte) (Response, error) {
	var root Root
	if err := json.Unmarshal(content, &root); err != nil {
		return Response{}, fmt.Errorf("error unmarshalling content %s: %w", string(content), err)
	}

	return root.Response, nil
}
