package admin

import (
	"database/sql"
	"time"
)

// Limit restricts the rows returned from a Query
// 0,10 would retrieve rows 1-10
// 5,10 would retrieve rows 6-15
type Limit struct {
	Offset uint // offset of the first row to return - initial row is 0 (not 1)
	Max    uint // maximum number of rows to return
}

type ProcessedLogEntry struct {
	CreatedAt       time.Time
	EventID         string
	RoutineName     string
	RoutineVersion  string
	RoutineInstance string
	MetaData        string
	Status          string
	ReferTable      string
	ReferID         string
}

func QueryProcessedLogsByReference(db *sql.DB, limit Limit, referid string, refertable, status string) (result []ProcessedLogEntry, err error) {

	var rows *sql.Rows
	rows, err = db.Query(`SELECT created_at, event_id, routine_name, routine_version,
routine_instance, meta_data, status, refer_table, refer_id
FROM eventz_subscriber_processed_logs
WHERE refer_id = ?
and refer_table = ?
and status = ?
order by created_at desc
LIMIT ?, ?`, referid, refertable, status, limit.Offset, limit.Max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var (
		createdAt          time.Time
		eventID            string
		routineName        string
		routineVersion     string
		routineInstance    string
		metaData           string
		matchingStatus     string
		matchingReferTable string
		matchingReferID    string
	)
	for rows.Next() {
		err = rows.Scan(&createdAt, &eventID, &routineName, &routineVersion, &routineInstance, &metaData, &matchingStatus, &matchingReferTable, &matchingReferID)
		if err != nil {
			return nil, err
		}
		result = append(result, ProcessedLogEntry{
			CreatedAt:       createdAt,
			EventID:         eventID,
			RoutineName:     routineName,
			RoutineVersion:  routineVersion,
			RoutineInstance: routineInstance,
			MetaData:        metaData,
			Status:          matchingStatus,
			ReferTable:      matchingReferTable,
			ReferID:         matchingReferID,
		})
	}

	return result, nil

}
