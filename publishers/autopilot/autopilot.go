package autopilot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/homemade/whiz/publishers"
	"github.com/segmentio/ksuid"
)

type autopilotRequest struct {
	Event     string      `json:"event"`
	ContactID string      `json:"contact_id"`
	ListID    string      `json:"list_id"`
	SegmentID string      `json:"segment_id"`
	Contact   interface{} `json:"contact"`
}

// {
// 	"event": "contact_added",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// 	"event": "contact_updated",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// {
// 	"event": "contact_unsubscribed",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// {
// 	"event": "contact_added_to_list",
// 	"list_id": "XXX",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// {
// 	"event": "contact_removed_from_list",
// 	"list_id": "XXX",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// {
// 	"event": "contact_entered_segment",
// 	"segment_id": "XXX",
// 	"list_id": "XXX",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

// {
// 	"event": "contact_left_segment",
// 	"segment_id": "XXX",
// 	"list_id": "XXX",
// 	"contact_id": "XXX",
// 	"contact": {
// 		...
// 	}
// }

func Parser(request publishers.WebhookRequest, secret string) (hook *publishers.Hook, err error) {

	var ar autopilotRequest
	err = json.Unmarshal(request.Body, &ar)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request %v", err)
	}

	// generate some ids (autopilot does not provide any)
	eventUUID := ksuid.New().String()
	eventID := fmt.Sprintf("autopilot_webhook:%s", eventUUID)
	// and a timestamp
	eventCreatedAt := time.Now()

	// TODO investigate how we can extract meaningful information for these values
	eventSource := "N/A" // default
	model := "N/A"       // default
	action := "N/A"      // default
	typ := "N/A"         // default
	userID := "N/A"      // default

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
