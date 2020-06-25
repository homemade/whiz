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

	// generate some Event ids (autopilot does not provide any)
	eventUUID := ksuid.New().String()
	eventID := fmt.Sprintf("autopilot_webhook:%s", eventUUID)
	// and a timestamp
	eventCreatedAt := time.Now()

	eventSource := p.Source
	// map the ContactID to UserID
	userID := ar.ContactID

	// populate Model, Type and Action based on the Autopilot event
	var model, action, typ string
	switch ar.Event {
	case "contact_added":
		model = "contact"
		action = "added"
		typ = ar.ContactID
	case "contact_updated":
		model = "contact"
		action = "updated"
		typ = ar.ContactID
	case "contact_unsubscribed":
		model = "contact"
		action = "unsubscribed"
		typ = ar.ContactID
	case "contact_added_to_list":
		model = "list"
		action = "added"
		typ = ar.ListID
	case "contact_removed_from_list":
		model = "list"
		action = "removed"
		typ = ar.ListID
	case "contact_entered_segment":
		model = "segment"
		action = "entered"
		typ = ar.SegmentID
	case "contact_left_segment":
		model = "segment"
		action = "left"
		typ = ar.SegmentID
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
		Type:           typ,
		Action:         action,
		UserID:         userID,
		ModelData:      string(modelData),
		SourceData:     string(sourceData),
	}, nil
}
