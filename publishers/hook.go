package publishers

import (
	"database/sql"
	"fmt"
	"time"
)

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

func Save(db *sql.DB, hook Hook) error {

	stmt, err := db.Prepare(`INSERT INTO eventz
(created_at,event_id,event_created_at,event_source,event_uuid,model,type,action,user_id,model_data,source_data)
VALUES(NOW(6),?,?,?,?,?,?,?,?,?,?);`)
	if err != nil {
		return fmt.Errorf("failed to prepare sql statement when saving hook with event id %s %v", hook.EventID, err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(hook.EventID, hook.EventCreatedAt, hook.EventSource, hook.EventUUID, hook.Model, hook.Type, hook.Action, hook.UserID, hook.ModelData, hook.SourceData)
	return err

}
