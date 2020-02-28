package models

import "time"

type EventzSubscriberProcessedLog struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	EventID string

	RoutineName     string
	RoutineVersion  string
	RoutineInstance string
	MetaData        string
	Status          string

	ReferEntity string
	ReferID     string
}
