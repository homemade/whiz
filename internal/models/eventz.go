package models

import (
	"time"
)

type Eventz struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	EventID        string
	EventCreatedAt time.Time
	EventSource    string
	EventUUID      string
	Model          string
	Type           string
	Action         string
	UserID         string
	ModelData      string
	SourceData     string
	SubGrp1Status  uint
	SubGrp2Status  uint
	SubGrp3Status  uint
	SubGrp4Status  uint
	SubGrp5Status  uint
	OnHold         uint

	ErrorCount uint
}

// Set Eventz table name to be `eventz`
func (Eventz) TableName() string {
	return "eventz"
}

func (e Eventz) GetEventSource() string {
	return e.EventSource
}

func (e Eventz) GetEventID() string {
	return e.EventID
}

func (e Eventz) GetEventCreatedAt() time.Time {
	return e.EventCreatedAt
}

func (e Eventz) GetEventUUID() string {
	return e.EventUUID
}

func (e Eventz) GetModelData() string {
	return e.ModelData
}

func (e Eventz) GetSourceData() string {
	return e.SourceData
}

func (e Eventz) GetModel() string {
	return e.Model
}

func (e Eventz) GetType() string {
	return e.Type
}

func (e Eventz) GetAction() string {
	return e.Action
}

func (e Eventz) GetUserID() string {
	return e.UserID
}

func (e Eventz) GetErrorCount() uint {
	return e.ErrorCount
}
