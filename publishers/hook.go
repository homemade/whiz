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
