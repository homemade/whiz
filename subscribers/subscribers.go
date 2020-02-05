package subscribers

import (
	"time"
)

type Registry interface {
	LoadRoutine(routine string, version string, instance string, metadata string, apimetrics APIMetrics) (Routine, error)
}

type APIMetrics interface {
	APIRequestResponse(requestURL string, responseStatusCode int)
}

type Routine interface {
	IsTemporaryError(err error) (is bool, status int)
	EventSources() []string
	EventLoopCeiling() int
	ProcessEvent(event Event) (Result, error)
	IsSkippableError(err error) bool
}

type Event interface {
	GetEventID() string
	GetEventCreatedAt() time.Time
	GetEventSource() string
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
	ReferTable() string
	ReferID() string
	Ignored() bool
}
