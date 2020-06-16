package publishers

import (
	"database/sql"
	"net/http"
)

type Parser func(r *http.Request, secret string) (hook *Hook, err error)

type WebhookPublisher struct {
	DB             *sql.DB
	Path           string
	Secret         string
	Parser         Parser
	AcceptedStatus int
}

func (p WebhookPublisher) Receive(w http.ResponseWriter, r *http.Request) (status int, err error) {

	var hook *Hook
	hook, err = p.Parser(r, p.Secret)
	if err != nil {
		return http.StatusBadRequest, err
	}

	writeString := func(status int, text string) error {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(status)
		_, err := w.Write([]byte(text))
		return err
	}

	if hook == nil { // handle requests that do not require an insert e.g. initial webhook validation requests
		err = writeString(p.AcceptedStatus, "")
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return status, nil
	}

	err = Save(p.DB, *hook)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = writeString(p.AcceptedStatus, "")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}
