package publishers

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Parser interface {
	Parse(request WebhookRequest, secret string) (hook *Hook, err error)
}

// The ParserFunc type is an adapter to allow the use of ordinary functions as Parsers.
// If f is a function with the appropriate signature, ParserFunc(f) is a Parser that returns f(request, secret).
type ParserFunc func(request WebhookRequest, secret string) (hook *Hook, err error)

// Parse returns f(request, secret).
func (f ParserFunc) Parse(request WebhookRequest, secret string) (hook *Hook, err error) {
	return f(request, secret)
}

type WebhookPublisher struct {
	db               *sql.DB
	parser           Parser
	secret           string
	statusCreated    int
	statusAccepted   int
	ignoreDuplicates bool
	errorAsserts     ErrorAsserts
}

func NewWebhookPublisher(db *sql.DB, parser Parser, options ...func(*WebhookPublisher)) WebhookPublisher {
	w := WebhookPublisher{
		db:             db,
		parser:         parser,
		statusCreated:  http.StatusCreated,
		statusAccepted: http.StatusAccepted,
	}
	for _, option := range options {
		option(&w)
	}
	return w
}

func WebhookPublisherSecret(secret string) func(*WebhookPublisher) {
	return func(w *WebhookPublisher) {
		w.secret = secret
	}
}

func WebhookPublisherStatusCreated(status int) func(*WebhookPublisher) {
	return func(w *WebhookPublisher) {
		w.statusCreated = status
	}
}

func WebhookPublisherStatusAccepted(status int) func(*WebhookPublisher) {
	return func(w *WebhookPublisher) {
		w.statusAccepted = status
	}
}

func WebhookPublisherIgnoreDuplicates(errorAsserts ErrorAsserts) func(*WebhookPublisher) {
	return func(w *WebhookPublisher) {
		w.ignoreDuplicates = true
		w.errorAsserts = errorAsserts
	}
}

type WebhookRequest struct {
	Path                  string
	HTTPMethod            string
	Headers               map[string][]string
	QueryStringParameters map[string][]string
	Body                  []byte
	BodyUnicode           string
}

func (p WebhookPublisher) Receive(w http.ResponseWriter, r *http.Request) (status int, err error) {

	path := ""
	query := make(map[string][]string)
	if r.URL != nil {
		path = r.URL.Path
		if r.URL.Query() != nil {
			query = r.URL.Query()
		}
	}
	var body []byte
	if r.Body != nil {
		body, err = ioutil.ReadAll(r.Body)
	}
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to read request %v", err)
	}
	r.Body.Close()

	var hook *Hook
	hook, err = p.parser.Parse(WebhookRequest{
		Path:                  path,
		HTTPMethod:            r.Method,
		Headers:               r.Header,
		Body:                  body,
		BodyUnicode:           string(body),
		QueryStringParameters: query,
	}, p.secret)
	if err != nil {
		return http.StatusBadRequest, err
	}

	writeString := func(status int, text string) error {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(status)
		_, err := w.Write([]byte(text))
		if err != nil {
			return fmt.Errorf("failed to write response %v", err)
		}
		return nil
	}

	if hook == nil { // handle requests that do not require an insert e.g. initial webhook validation requests
		status = p.statusAccepted
		err = writeString(status, "")
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return status, nil
	}

	err = Save(p.db, *hook)
	if err == nil {
		status = p.statusCreated
	} else {
		if p.ignoreDuplicates && p.errorAsserts.IsDuplicateDatabaseEntry(err) {
			status = p.statusAccepted
		} else {
			return http.StatusInternalServerError, err
		}
	}

	err = writeString(status, "")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}
