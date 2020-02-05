package publishers

import (
	"database/sql"
	"fmt"
	"io"
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

	// HTMLBlob sends an HTTP blob response with status code.
	HTMLBlob(code int, b []byte) error

	// String sends a string response with status code.
	String(code int, s string) error

	// JSON sends a JSON response with status code.
	JSON(code int, i interface{}) error

	// JSONPretty sends a pretty-print JSON with status code.
	JSONPretty(code int, i interface{}, indent string) error

	// JSONBlob sends a JSON blob response with status code.
	JSONBlob(code int, b []byte) error

	// JSONP sends a JSONP response with status code. It uses `callback` to construct
	// the JSONP payload.
	JSONP(code int, callback string, i interface{}) error

	// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
	// to construct the JSONP payload.
	JSONPBlob(code int, callback string, b []byte) error

	// XML sends an XML response with status code.
	XML(code int, i interface{}) error

	// XMLPretty sends a pretty-print XML with status code.
	XMLPretty(code int, i interface{}, indent string) error

	// XMLBlob sends an XML blob response with status code.
	XMLBlob(code int, b []byte) error

	// Blob sends a blob response with status code and content type.
	Blob(code int, contentType string, b []byte) error

	// Stream sends a streaming response with status code and content type.
	Stream(code int, contentType string, r io.Reader) error

	// File sends a response with the content of the file.
	File(file string) error

	// Attachment sends a response as attachment, prompting client to save the
	// file.
	Attachment(file string, name string) error

	// Inline sends a response as inline, opening the file in the browser.
	Inline(file string, name string) error

	// NoContent sends a response with no body and a status code.
	NoContent(code int) error
}
