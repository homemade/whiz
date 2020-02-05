package raisely

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/homemade/whiz/publishers"
)

type WebhookPublisher struct {
	DB *sql.DB
}

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

func (p WebhookPublisher) Path() string {
	return "raisely"
}

func (p WebhookPublisher) Receive(request publishers.Request, response publishers.Response, logger publishers.Logger) (status int, err error) {

	var hook *publishers.Hook
	hook, err = ParseRaisleyRequest(request)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to parse request `%s` from path %s %v", request.BodyUnicode, request.Path, err)
	}
	if hook == nil { // handle requests that do not require an insert e.g. initial raisely webhook validation
		status = http.StatusAccepted
		err = response.String(status, "")
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return status, nil
	}

	err = publishers.Save(p.DB, *hook)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	status = http.StatusAccepted
	err = response.String(status, "")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}

func ParseRaisleyRequest(r publishers.Request) (hook *publishers.Hook, err error) {
	if len(r.Body) < 3 {
		return nil, nil // handle initial empty request from raisely of `{}` - used to validate webhook
	}
	// required fields
	source_data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source_data %v", err)
	}
	var rr raisleyRequest
	err = json.Unmarshal(r.Body, &rr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request %v", err)
	}
	event_id := fmt.Sprintf("raisely_webhook:%s", rr.Event.UUID)
	var event_created_at time.Time
	event_created_at, err = time.Parse(time.RFC3339, rr.Event.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event_created_at %v", err)
	}
	event_source := rr.Event.Source
	event_uuid := rr.Event.UUID

	parts := strings.Split(rr.Event.Type, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to parse model and action, expected 2 parts seperated by . but have %d")
	}
	model := parts[0]
	action := parts[1]
	// set type and user id as appropriate for the model
	typ := "N/A"     // default
	user_id := "N/A" // default
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
				user_id = s
			}
		}
	}
	var model_data []byte
	model_data, err = json.Marshal(rr.Event.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model_data %v", err)
	}
	md := string(model_data)
	sd := string(source_data)

	return &publishers.Hook{
		EventID:        event_id,
		EventCreatedAt: event_created_at,
		EventSource:    event_source,
		EventUUID:      event_uuid,
		Model:          model,
		Type:           typ,
		Action:         action,
		UserID:         user_id,
		ModelData:      md,
		SourceData:     sd,
	}, nil
}
