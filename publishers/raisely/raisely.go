package raisely

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/homemade/whiz/publishers"
)

type raisleyRequest struct {
	Secret string `json:"secret"`
	Event  struct {
		UUID      string      `json:"uuid"`
		Type      string      `json:"type"`
		CreatedAt string      `json:"createdAt"`
		Source    string      `json:"source"`
		Data      interface{} `json:"data"`
	} `json:"data"`
}

func Parse(request publishers.WebhookRequest, secret string) (hook *publishers.Hook, err error) {

	if len(request.Body) < 3 {
		return nil, nil // handle initial empty request from raisely of `{}` - used to validate webhook
	}

	var rr raisleyRequest
	err = json.Unmarshal(request.Body, &rr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request %v", err)
	}

	if secret != "" {
		if secret != rr.Secret {
			return nil, errors.New("invalid secret")
		}
	}

	eventID := fmt.Sprintf("raisely_webhook:%s", rr.Event.UUID)
	var eventCreatedAt time.Time
	eventCreatedAt, err = time.Parse(time.RFC3339, rr.Event.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EventCreatedAt %v", err)
	}
	eventSource := rr.Event.Source
	eventUUID := rr.Event.UUID

	parts := strings.Split(rr.Event.Type, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to parse model and action, expected 2 parts seperated by . but have %d", len(parts))
	}
	model := parts[0]
	action := parts[1]
	// set type and user id as appropriate for the model
	typ := "N/A"    // default
	userID := "N/A" // default
	if m, ok := rr.Event.Data.(map[string]interface{}); ok {
		// first set type
		field := "type"      // for most models we look for the type field
		if model == "user" { // but for users we use the permission field instead
			field = "permission"
		}
		if t, exists := m[field]; exists {
			if s, tok := t.(string); tok {
				typ = s
			}
		}
		// then set user
		field = "userUuid"   // for most models we look for the userUuid field
		if model == "user" { // but for users we use the uuid field instead
			field = "uuid"
		}
		if u, exists := m[field]; exists {
			if s, uok := u.(string); uok {
				userID = s
			}
		}
	}

	var sourceData []byte
	sourceData, err = json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SourceData %v", err)
	}
	var modelData []byte
	modelData, err = json.Marshal(rr.Event.Data)
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
