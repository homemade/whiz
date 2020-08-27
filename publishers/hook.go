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

	d := fmt.Sprintf("%T", db.Driver())

	s := `INSERT INTO eventz
(created_at,event_id,event_created_at,event_source,event_uuid,model,type,action,user_id,model_data,source_data)`
	switch d {
	case "*mysql.MySQLDriver":
		s = s + " VALUES(NOW(6),?,?,?,?,?,?,?,?,?,?);"
	case "*pq.Driver":
		s = s + " VALUES(NOW(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10);"
	default:
		return fmt.Errorf("unsupported database driver %s", d)
	}

	stmt, err := db.Prepare(s)
	if err != nil {
		return fmt.Errorf("failed to prepare sql statement when saving hook with event id %s %v", hook.EventID, err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(hook.EventID, hook.EventCreatedAt, hook.EventSource, hook.EventUUID, hook.Model, hook.Type, hook.Action, hook.UserID, hook.ModelData, hook.SourceData)
	return err

}
