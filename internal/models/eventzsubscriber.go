package models

import (
	"fmt"
	"time"
)

type EventzSubscriber struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	Name        string
	Version     string
	Instance    string
	MetaData    string
	LastRunAt   *time.Time
	Group       uint
	Priority    uint
	LastErrorAt *time.Time
}

func (s EventzSubscriber) Desc() string {
	return fmt.Sprintf("Group: %d Routine: %s %s %s Priority: %d", s.Group, s.Name, s.Version, s.Instance, s.Priority)
}
