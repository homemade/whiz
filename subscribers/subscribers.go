package subscribers

import (
	"time"
)

type Definition struct {
	Name     string
	Version  string
	Instance string
	MetaData string
}

type Registry interface {
	LoadRoutine(definition Definition) (Routine, error)
}

type Routine interface {
	EventSources() []string
	EventLoopCeiling() int
	ProcessEvent(event Event) (Result, error)
	AssertError(err error) (temporary bool, status int)
	CauseOfError(err error) error
}

type Event interface {
	GetEventID() string
	GetEventCreatedAt() time.Time
	GetEventSource() string
	GetEventSourceConfig() string
	GetEventUUID() string
	GetModelData() string
	GetSourceData() string
	GetModel() string
	GetType() string
	GetAction() string
	GetUserID() string
	GetErrorCount() uint
}

type Result interface {
	MetaData() string
	Status() string
	ReferEntity() string
	ReferID() string
	Ignored() bool
}
