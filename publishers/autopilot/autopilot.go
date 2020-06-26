package autopilot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/homemade/whiz/publishers"
	"github.com/segmentio/ksuid"
)

type Parser struct {
	Source string
}

type autopilotRequest struct {
	Event     string      `json:"event"`
	ContactID string      `json:"contact_id"`
	ListID    string      `json:"list_id"`
	SegmentID string      `json:"segment_id"`
	Contact   interface{} `json:"contact"`
}

func (p Parser) Parse(request publishers.WebhookRequest, secret string) (hook *publishers.Hook, err error) {

	var ar autopilotRequest
	err = json.Unmarshal(request.Body, &ar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request %v", err)
	}

	// generate Event ID (autopilot does not provide one)
	eventID := fmt.Sprintf("autopilot_webhook:%s", ksuid.New().String())
	// and a timestamp
	eventCreatedAt := time.Now()

	eventSource := p.Source

	// map the ContactID to UserID
	userID := ar.ContactID

	// populate Model, EventUUID and Action based on the autopilot event
	var model, eventUUID, action string
	switch ar.Event {
	case "contact_added":
		model = "contact"
		eventUUID = ar.ContactID
		action = "added"
	case "contact_updated":
		model = "contact"
		eventUUID = ar.ContactID
		action = "updated"
	case "contact_unsubscribed":
		model = "contact"
		eventUUID = ar.ContactID
		action = "unsubscribed"
	case "contact_added_to_list":
		model = "list"
		eventUUID = ar.ListID
		action = "added"
	case "contact_removed_from_list":
		model = "list"
		eventUUID = ar.ListID
		action = "removed"
	case "contact_entered_segment":
		model = "segment"
		eventUUID = ar.SegmentID
		action = "entered"
	case "contact_left_segment":
		model = "segment"
		eventUUID = ar.SegmentID
		action = "left"
	default:
		return nil, fmt.Errorf("unsupported autopilot event %s", ar.Event)
	}

	var sourceData []byte
	sourceData, err = json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SourceData %v", err)
	}
	var modelData []byte
	modelData, err = json.Marshal(ar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ModelData %v", err)
	}

	return &publishers.Hook{
		EventID:        eventID,
		EventCreatedAt: eventCreatedAt,
		EventSource:    eventSource,
		EventUUID:      eventUUID,
		Model:          model,
		Type:           "", // not required for autopilot
		Action:         action,
		UserID:         userID,
		ModelData:      string(modelData),
		SourceData:     string(sourceData),
	}, nil
}
