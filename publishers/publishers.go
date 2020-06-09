package publishers

import (
	"database/sql"
	"fmt"
	"time"
)

type Request struct {
	Path                  string
	HTTPMethod            string
	Headers               map[string][]string
	QueryStringParameters map[string][]string
	Body                  []byte
	BodyUnicode           string
	BodyReadErr           error
}

type Hook struct {
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
}

func Save(db *sql.DB, hook Hook) (err error) {

	var stmt *sql.Stmt
	stmt, err = db.Prepare(`INSERT INTO eventz
(created_at,event_id,event_created_at,event_source,event_uuid,model,type,action,user_id,model_data,source_data)
VALUES(NOW(6),?,?,?,?,?,?,?,?,?,?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare sql statement when saving hook with event id %s %v", hook.EventID, err)
	}
	defer func() {
		err = stmt.Close()
		if err != nil {
			err = fmt.Errorf("failed to close sql statement when saving hook with event id %s %v", hook.EventID, err)
		}
	}()
	_, err = stmt.Exec(hook.EventID, hook.EventCreatedAt, hook.EventSource, hook.EventUUID, hook.Model, hook.Type, hook.Action, hook.UserID, hook.ModelData, hook.SourceData)
	if err != nil {
		return fmt.Errorf("failed to execute sql statement when saving hook with event id %s %v", hook.EventID, err)
	}
	return err

}

type Logger interface {
	Warn(i ...interface{})
	Warnf(format string, args ...interface{})
}

type Publisher interface {
	Path() string
	Receive(request Request, response Response, logger Logger) (status int, err error)
}

type Response interface {

	// HTML sends an HTTP response with status code.
	HTML(code int, html string) error

	// String sends a string response with status code.
	String(code int, s string) error

	// JSON sends a JSON response with status code.
	JSON(code int, i interface{}) error

	// XML sends an XML response with status code.
	XML(code int, i interface{}) error

	// NoContent sends a response with no body and a status code.
	NoContent(code int) error
}
