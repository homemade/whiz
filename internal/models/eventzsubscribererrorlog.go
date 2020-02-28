package models

import (
	"time"
)

type EventzSubscriberErrorLog struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	EventID string

	RoutineName     string
	RoutineVersion  string
	RoutineInstance string
	ErrorCode       uint
	ErrorMessage    string
	Status          string

	ReferEntity string
	ReferID     string
}
