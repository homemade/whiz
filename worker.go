package whiz

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/homemade/whiz/internal/models"
	"github.com/homemade/whiz/sqlbuilder"
	"github.com/homemade/whiz/subscribers"
)

type WorkerStatus struct {
	Started         time.Time
	PollingInterval time.Duration
	LastPolled      time.Time
	LastRun         time.Time
	DBRW            string
	DBRO            string
}

type SubscriberEvents []models.Eventz

func (se SubscriberEvents) Len() int {
	return len(se)
}
func (se SubscriberEvents) Swap(i, j int) {
	se[i], se[j] = se[j], se[i]
}
func (se SubscriberEvents) Less(i, j int) bool {
	return se[j].EventCreatedAt.After(se[i].EventCreatedAt)
}

type Worker struct {
	dbRW            *sql.DB
	dbRO            *sql.DB
	reg             subscribers.Registry
	loggerOutput    io.Writer
	lockName        string
	pollingInterval time.Duration
	externalPolling bool
	errorCeiling    time.Duration
	tick            *time.Ticker
	mu              *sync.Mutex
	active          bool
	status          *WorkerStatus
	shutdown        bool
}

type cancelledFunc func() bool

type eventLoopParams struct {
	Started      time.Time
	DBRW         *sql.DB
	DBRO         *sql.DB
	LoggerOutput io.Writer
	LockName     string
	ErrorCeiling time.Duration
	Cancelled    cancelledFunc
	Registry     subscribers.Registry
}

type ErrorWithCause interface {
	error
	Cause() error
}

type errWithCause struct {
	err   string
	cause error
}

func (e errWithCause) Error() string {
	return e.err
}

func (e errWithCause) Cause() error {
	return e.cause
}

func NewWorker(dbRW *sql.DB, reg subscribers.Registry, loggeroutput io.Writer, options ...func(*Worker) error) (*Worker, error) {
	lockName := "whiz-worker"          // default
	pollingInterval := 1 * time.Minute // default
	errorCeiling := 1 * time.Minute    // default
	w := Worker{
		dbRW:            dbRW,
		dbRO:            dbRW, // default
		reg:             reg,
		loggerOutput:    loggeroutput,
		lockName:        lockName,
		pollingInterval: pollingInterval,
		errorCeiling:    errorCeiling,
		mu:              &sync.Mutex{},
		status: &WorkerStatus{
			DBRW: fmt.Sprintf("%T", dbRW.Driver()),
		},
	}
	for _, option := range options {
		if err := option(&w); err != nil {
			return nil, err

		}
	}
	return &w, nil
}

func LockName(s string) func(*Worker) error {
	return func(w *Worker) error {
		w.lockName = s
		return nil
	}
}

func PollingInterval(t time.Duration) func(*Worker) error {
	return func(w *Worker) error {
		w.pollingInterval = t
		return nil
	}
}

func ExternalPolling() func(*Worker) error {
	return func(w *Worker) error {
		w.externalPolling = true
		return nil
	}
}

func ErrorCeiling(t time.Duration) func(*Worker) error {
	return func(w *Worker) error {
		w.errorCeiling = t
		return nil
	}
}

// option for providing a read replica
func ReadOnlyDatabase(dbRO *sql.DB) func(*Worker) error {
	return func(w *Worker) error {
		w.dbRO = dbRO
		w.status.DBRO = fmt.Sprintf("%T", dbRO.Driver())
		return nil
	}
}

func (w *Worker) pollingLoop(count int) {
	i := count
	for t := range w.tick.C {
		w.status.LastPolled = t
		if w.active {
			logInfo(w.loggerOutput, fmt.Sprintf("Worker cancelled next poll as previous run still active %v", w.status))
			continue // handle runs that take longer than the event loop polling interval
		}
		w.status.LastRun = t
		i = i + 1
		logMetrics(w.loggerOutput, "worker_poll", i, time.Since(w.status.Started).Seconds())
		w.nextRun(t)
		// handle changes to polling interval whilst running
		if w.pollingInterval != w.status.PollingInterval {
			logInfo(w.loggerOutput, fmt.Sprintf("Worker restarting polling as interval has changed from %v to %v", w.status.PollingInterval, w.pollingInterval))
			w.tick.Stop()
			w.tick = time.NewTicker(w.pollingInterval)
			w.status.PollingInterval = w.pollingInterval
			w.pollingLoop(i)
		}
	}
}

func (w *Worker) StartPolling() {
	if w.externalPolling {
		logError(w.loggerOutput, errors.New("Worker has been configured to use external polling, call PollNow() to trigger next run"))
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("Worker returned from call to start polling with error %v", r))
		}
	}()
	w.status.Started = time.Now()
	w.status.PollingInterval = w.pollingInterval
	w.tick = time.NewTicker(w.pollingInterval)
	go func() {
		w.pollingLoop(0)
	}()
	logInfo(w.loggerOutput, "Worker started polling")
}

func (w *Worker) PollNow() {
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("Worker returned from call to poll now with error %v", r))
		}
	}()
	t := time.Now()
	w.status.LastPolled = t
	if w.active {
		logInfo(w.loggerOutput, fmt.Sprintf("Worker cancelled poll now request as previous run still active %v", w.status))
		return // already polling
	}
	w.status.LastRun = t
	logMetrics(w.loggerOutput, "worker_poll_now", 1, time.Since(t).Seconds())
	w.nextRun(t)
}

func (w *Worker) ReProcessSubscriberEvent(eventid string, timeout int) {
	t := time.Now()
	// wait until previous polling is no longer active or timeout hit
	for w.active && time.Now().Before(t.Add(time.Second*time.Duration(timeout))) {
		time.Sleep(500 * time.Millisecond)
	}
	if w.active {
		logInfo(w.loggerOutput, fmt.Sprintf("Worker cancelled call to reprocess event as previous run still active after %s seconds %v", timeout, w.status))
		return //timeout
	}
	// reset the event
	err := resetSubscriberEvent(w.dbRW, eventid)
	if err != nil {
		logError(w.loggerOutput, errWithCause{fmt.Sprintf("failed to reset on hold record matching Event#%s", eventid), err})
		return
	}
	logInfo(w.loggerOutput, fmt.Sprintf("Worker has reset on hold Event#%s and is making a request to poll now", eventid))
	// trigger polling
	w.PollNow()
}

func (w Worker) Status() WorkerStatus {
	return *w.status
}

func (w *Worker) nextRun(t time.Time) {
	// ensure only one background worker can run at a time using an in memory check
	// we also use a database lock in case we accidently have multiple instances deployed see runEventLoops()
	w.mu.Lock()
	w.active = true
	w.mu.Unlock()
	defer func(wrk *Worker) {
		wrk.mu.Lock()
		wrk.active = false
		wrk.mu.Unlock()
	}(w)
	runEventLoops(eventLoopParams{
		Started:      t,
		DBRW:         w.dbRW,
		DBRO:         w.dbRO,
		LoggerOutput: w.loggerOutput,
		LockName:     w.lockName,
		ErrorCeiling: w.errorCeiling,
		Cancelled: cancelledFunc(func() bool {
			return w.shutdown
		}),
		Registry: w.reg,
	})
}

func (w *Worker) Shutdown() {
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("Worker returned from call to shutdown with error %v", r))
		}
	}()
	if w.tick != nil {
		w.tick.Stop()
	}
	w.shutdown = true
	for w.active {
		// wait for current event loop to finish
	}
}

func runEventLoops(p eventLoopParams) {

	// NOTE: use read only db where at all possible in the worker (this frees up the read write db for the web server)

	// we also need to handle long running processes again to ensure only 1 process is ever running
	// we use a simple database table for this
	var err error
	var locked int64
	var result sql.Result

	var s string
	s, err = sqlbuilder.DriverSpecificSQL{
		MySQL:    "INSERT INTO eventz_locks (name,created_at) VALUES(?,NOW(6));",
		Postgres: "INSERT INTO eventz_locks (name,created_at) VALUES($1,NOW());",
	}.Switch(p.DBRW)
	if err != nil {
		logInfo(p.LoggerOutput, fmt.Sprintf("next event loops run failed %v", err))
		return
	}
	if result, err = p.DBRW.Exec(s, p.LockName); err == nil {
		locked, err = result.RowsAffected()
	}
	if err == nil && locked != 1 {
		err = fmt.Errorf("%d locks inserted for %s", locked, p.LockName)
	}
	if err != nil { // failed to acquire lock - another process is still running so just return
		logInfo(p.LoggerOutput, fmt.Sprintf("next event loops run cancelled as detected another process is running with lock %s, detected with %v", p.LockName, err))
		return
	}
	defer func() {
		s, err = sqlbuilder.DriverSpecificSQL{
			MySQL:    "DELETE FROM eventz_locks WHERE name = ?;",
			Postgres: "DELETE FROM eventz_locks WHERE name = $1;",
		}.Switch(p.DBRW)
		var unlocked int64
		if err == nil {
			if result, err = p.DBRW.Exec(s, p.LockName); err == nil {
				unlocked, err = result.RowsAffected()
			}
		}
		if err == nil && unlocked != 1 {
			err = fmt.Errorf("%d locks deleted for %s", unlocked, p.LockName)
		}
		if err != nil {
			logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to release event loops lock %s during clean exit", p.LockName), err})
		}
		return
	}()

	// load event sources
	// we do this on each run to pickup any changes
	var eventSources map[string]models.EventzSource
	eventSources, err = readEventSourcesFromDB(p.DBRO)
	if err != nil {
		logError(p.LoggerOutput, errWithCause{"failed to read event sources %v", err})
		return
	}

	// run all the subscriber event loops
	z := 5                // (we have up to 5 groups of subscribers)
	var wg sync.WaitGroup // and a wait group so we know when they have all completed
	wg.Add(z)
	// then run each concurrently
	var i uint
	for i = 1; i <= 5; i++ {
		go func(grp uint) {
			var err error
			defer func() {
				wg.Done()
				if err != nil {
					logInfo(p.LoggerOutput, fmt.Sprintf("Worker subscriber group %d done with error %v", grp, err))
				} else {
					if r := recover(); r != nil {
						logInfo(p.LoggerOutput, fmt.Sprintf("Worker subscriber group %d done with unhandled error %v", grp, r))
					} else {
						logInfo(p.LoggerOutput, fmt.Sprintf("Worker subscriber group %d done", grp))
					}
				}
			}()
			// NOTE: use read only db where at all possible in the worker (this frees up the read/write db for the web server)
			var dbRoutines []models.EventzSubscriber
			dbRoutines, err = readRoutinesFromDB(p.DBRO, grp, p.ErrorCeiling)
			if err != nil {
				logError(p.LoggerOutput, errWithCause{"failed to read dbRoutines", err})
				return
			}
			logMetrics(p.LoggerOutput, fmt.Sprintf("worker_routines:%d", grp), len(dbRoutines), time.Since(p.Started).Seconds())
			for _, r := range dbRoutines {
				var sr subscribers.Routine
				sr, err = p.Registry.LoadRoutine(subscribers.Definition{
					Name:     r.Name,
					Version:  r.Version,
					Instance: r.Instance,
					MetaData: r.MetaData,
				})
				if err != nil {
					logError(p.LoggerOutput, errWithCause{"failed to load subscriber routine", err})
					return
				}
				err = updateLastRunAtInDB(p.DBRW, r)
				if err != nil {
					logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to update last_run_at during execution of %s", r.Desc()), err})
					return
				}
				var events SubscriberEvents
				events, err = readSubscriberEventsFromDB(p.DBRO, r.Group, r.Priority, sr.EventSources(), sr.EventLoopCeiling())
				if err != nil {
					logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to read subscriber events during execution of %s", r.Desc()), err})
					return
				}
				logMetrics(p.LoggerOutput, fmt.Sprintf("worker_events:%d.%d", r.Group, r.Priority), len(events), time.Since(p.Started).Seconds())
				processedEvents := 0
				for _, e := range events {
					// handle cancellation requests
					if p.Cancelled() {
						logInfo(p.LoggerOutput, fmt.Sprintf("Worker subscriber group %d exiting event loop due to cancellation request", grp))
						break
					}
					// add any event source config
					if eventSource, exists := eventSources[e.EventSource]; exists {
						e.EventSourceConfig = eventSource.Config
					}
					var procResult subscribers.Result
					procResult, err = sr.ProcessEvent(e)
					if err != nil {
						temporaryError, skippableError, status := sr.AssertError(err)
						if !temporaryError { // we don't put events on hold for temporary errors - we will try again later
							if dberr := markSubscriberEventAsOnHold(p.DBRW, e.EventID); dberr != nil {
								logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to mark event#%s as onhold", e.EventID), dberr})
							}
						}
						// augment the error with extra details
						msg := fmt.Sprintf("error running %s %v", r.Desc(), err)
						if e.EventID != "" {
							msg = fmt.Sprintf("%s for user #%s", msg, e.UserID)
						}
						if e.EventID != "" {
							msg = fmt.Sprintf("%s when processing event #%s", msg, e.EventID)
						}
						// and log it
						logError(p.LoggerOutput, errWithCause{msg, err})
						if err2 := updateLastErrorAtInDB(p.DBRW, r); err2 != nil {
							logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to update last_error_at during execution of %s", r.Desc()), err2})
						}
						// try and also add error to the log table
						insertIntoErrorLog(p.LoggerOutput, p.DBRW, r, err, status, e.EventID, procResult)

						// dont break out of the loop for skippable errors - these will be left on hold
						if !skippableError {
							break
						}

					} else {
						processedEvents = processedEvents + 1
						err = markSubscriberEventAsProcessed(p.DBRW, r, e, procResult)
						if err != nil {
							logError(p.LoggerOutput, errWithCause{fmt.Sprintf("failed to mark event#%s as processed", e.EventID), err})
							return
						}
					}
				}
				logMetrics(p.LoggerOutput, fmt.Sprintf("worker_processed:%d.%d", r.Group, r.Priority), processedEvents, time.Since(p.Started).Seconds())
				if err != nil {
					return
				}
			}
		}(i)
	}
	// wait until all are complete
	wg.Wait()

	logMetrics(p.LoggerOutput, "worker_execution", 1, time.Since(p.Started).Seconds())

}

func readEventSourcesFromDB(db *sql.DB) (map[string]models.EventzSource, error) {
	result := make(map[string]models.EventzSource)
	var (
		name   string
		config string
	)
	rows, err := db.Query("SELECT name, config FROM eventz_sources;")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&name, &config)
		if err != nil {
			return result, err
		}
		result[name] = models.EventzSource{
			Name:   name,
			Config: config,
		}
	}
	return result, nil
}

func readRoutinesFromDB(db *sql.DB, group uint, errceiling time.Duration) ([]models.EventzSubscriber, error) {

	result := make([]models.EventzSubscriber, 0)

	routinesSQL := "SELECT name, version, instance, meta_data, priority FROM eventz_subscribers WHERE eventz_subscribers.group = %d AND " +
		"priority > 0 AND eventz_subscribers.group IN (SELECT eventz_subscribers.group FROM eventz_subscribers GROUP BY eventz_subscribers.group " +
		"HAVING MAX(last_error_at) IS NULL OR MAX(last_error_at) < "

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    " DATE_SUB(NOW(6),INTERVAL %f SECOND)) ORDER BY priority ASC;",
		Postgres: " NOW() - INTERVAL '%f SECOND') ORDER BY priority ASC;",
	}.Switch(db)
	if err != nil {
		return result, err
	}

	routinesSQL = fmt.Sprintf(routinesSQL+s, group, errceiling.Seconds())

	var (
		rows      *sql.Rows
		name      string
		version   string
		instance  string
		meta_data sql.NullString
		priority  uint
	)
	rows, err = db.Query(routinesSQL)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&name, &version, &instance, &meta_data, &priority)
		if err != nil {
			return result, err
		}
		result = append(result, models.EventzSubscriber{
			Group:    group,
			Name:     name,
			Version:  version,
			Instance: instance,
			MetaData: meta_data.String,
			Priority: priority,
		})
	}
	return result, nil
}

func updateLastRunAtInDB(dbRW *sql.DB, sub models.EventzSubscriber) error {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    " NOW(6) WHERE name = ? AND version = ? AND instance = ?;",
		Postgres: " NOW() WHERE name = $1 AND version = $2 AND instance = $3;",
	}.Switch(dbRW)
	if err != nil {
		return err
	}
	s = "UPDATE eventz_subscribers SET last_run_at = " + s

	var result sql.Result
	result, err = dbRW.Exec(s, sub.Name, sub.Version, sub.Instance)
	if err != nil {
		return err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row to be affected but %d were", rowsAffected)
	}
	return nil
}

func updateLastErrorAtInDB(dbRW *sql.DB, sub models.EventzSubscriber) error {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    "NOW(6) WHERE name = ? AND version = ? AND instance = ?;",
		Postgres: "NOW() WHERE name = $1 AND version = $2 AND instance = $3;",
	}.Switch(dbRW)
	if err != nil {
		return err
	}
	s = "UPDATE eventz_subscribers SET last_error_at = " + s
	var result sql.Result
	result, err = dbRW.Exec(s, sub.Name, sub.Version, sub.Instance)
	if err != nil {
		return err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row to be affected but %d were", rowsAffected)
	}
	return nil
}

func insertIntoErrorLog(loggerOutput io.Writer, dbRW *sql.DB, sub models.EventzSubscriber, e error, status int, eventid string, res subscribers.Result) {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    "VALUES(?,?,?,?,?,?,?,?,?,NOW(6));",
		Postgres: "VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW());",
	}.Switch(dbRW)
	if err != nil {
		logError(loggerOutput, err)
		return
	}
	s = `INSERT INTO eventz_subscriber_error_logs
	(event_id,routine_name,routine_version,routine_instance,error_code,error_message,status,refer_entity,refer_id,created_at)` + s

	var stmt *sql.Stmt
	stmt, err = dbRW.Prepare(s)
	if err != nil {
		logError(loggerOutput, err)
		return
	}
	defer func() {
		err = stmt.Close()
		if err != nil {
			logError(loggerOutput, err)
			return
		}
	}()
	var cause error
	if ec, ok := e.(ErrorWithCause); ok {
		cause = ec.Cause()
	}
	_, err = stmt.Exec(eventid, sub.Name, sub.Version, sub.Instance, status, fmt.Sprintf("Error: %v \nCause: %v", e, cause), res.Status(), res.ReferEntity(), res.ReferID())
	if err != nil {
		logError(loggerOutput, err)
		return
	}
	return
}

func readSubscriberEventsFromDB(db *sql.DB, group uint, priority uint, sources []string, ceiling int) (SubscriberEvents, error) {
	result := make(SubscriberEvents, 0)

	if group < 1 || group > 5 {
		return result, fmt.Errorf("invalid group %d", group)
	}

	if sources == nil {
		return result, errors.New("missing sources")
	}
	if len(sources) < 1 {
		return result, errors.New("invalid sources")
	}

	var (
		eventID         string
		eventCreatedAt  time.Time
		eventSource     sql.NullString
		eventUuid       sql.NullString
		eventModel      string
		eventType       string
		eventAction     string
		eventUserID     sql.NullString
		eventModelData  sql.NullString
		eventSourceData sql.NullString
	)

	subscriberEventsSQL := fmt.Sprintf(`SELECT event_id, event_created_at, event_source, event_uuid, model, type, action, user_id, model_data, source_data
FROM eventz
WHERE on_hold = 0
AND sub_grp%d_status = (%d - 1)`, group, priority)

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    " AND event_created_at < DATE_SUB(NOW(6),INTERVAL %d MINUTE)",
		Postgres: " AND event_created_at < NOW() - INTERVAL '%d MINUTE'",
	}.Switch(db)
	if err != nil {
		return result, err
	}

	// sources can be a wild card
	if sources[0] == "*" {
		subscriberEventsSQL = subscriberEventsSQL + fmt.Sprintf(s+" ORDER BY event_created_at,event_id ASC;", ceiling)
	} else { // or a filter
		eventSources := strings.Join(strings.Fields(fmt.Sprint(sources)), "','")
		eventSources = strings.Replace(eventSources, "[", "('", 1)
		eventSources = strings.Replace(eventSources, "]", "')", 1)
		subscriberEventsSQL = subscriberEventsSQL + fmt.Sprintf(" AND event_source IN %s "+s+" ORDER BY event_created_at,event_id ASC;", eventSources, ceiling)
	}
	var rows *sql.Rows
	rows, err = db.Query(subscriberEventsSQL)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&eventID, &eventCreatedAt, &eventSource, &eventUuid, &eventModel, &eventType, &eventAction, &eventUserID, &eventModelData, &eventSourceData)
		if err != nil {
			return result, err
		}
		// calculate error count for the event
		var errorCount uint
		row := db.QueryRow(fmt.Sprintf("SELECT COUNT(1) FROM eventz_subscriber_error_logs WHERE event_id = '%s';", eventID))
		err = row.Scan(&errorCount)
		if err != nil && err != sql.ErrNoRows {
			return result, err
		}
		nextEvent := models.Eventz{
			EventID:        eventID,
			EventCreatedAt: eventCreatedAt,
			Model:          eventModel,
			Type:           eventType,
			Action:         eventAction,
			ErrorCount:     errorCount,
		}
		if eventModelData.Valid {
			nextEvent.ModelData = eventModelData.String
		}
		if eventSourceData.Valid {
			nextEvent.SourceData = eventSourceData.String
		}
		if eventSource.Valid {
			nextEvent.EventSource = eventSource.String
		}
		if eventUuid.Valid {
			nextEvent.EventUUID = eventUuid.String
		}
		if eventUserID.Valid {
			nextEvent.UserID = eventUserID.String
		}
		result = append(result, nextEvent)
	}
	return result, nil
}

func markSubscriberEventAsProcessed(dbRW *sql.DB, sub models.EventzSubscriber, event models.Eventz, res subscribers.Result) error {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    "UPDATE eventz SET sub_grp%d_status = ? WHERE event_id = ?;",
		Postgres: "UPDATE eventz SET sub_grp%d_status = $1 WHERE event_id = $2;",
	}.Switch(dbRW)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf(s, sub.Group)
	var result sql.Result
	result, err = dbRW.Exec(stmt, sub.Priority, event.EventID)
	if err != nil {
		return err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row to be affected in eventz table but %d were", rowsAffected)
	}

	if !res.Ignored() {

		s, err = sqlbuilder.DriverSpecificSQL{
			MySQL:    "VALUES(?,?,?,?,?,?,?,?,NOW(6));",
			Postgres: "VALUES($1,$2,$3,$4,$5,$6,$7,$8,NOW());",
		}.Switch(dbRW)
		if err != nil {
			return err
		}

		stmt = `INSERT INTO eventz_subscriber_processed_logs
		(event_id,routine_name,routine_version,routine_instance,meta_data,status,refer_entity,refer_id,created_at)` + s
		result, err = dbRW.Exec(stmt, event.EventID, sub.Name, sub.Version, sub.Instance, res.MetaData(), res.Status(), res.ReferEntity(), res.ReferID())
		if err != nil {
			return err
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row to be affected in eventz_subscriber_processed_logs table but %d were", rowsAffected)
		}
	}

	return nil
}

func markSubscriberEventAsOnHold(dbRW *sql.DB, eventid string) error {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    "UPDATE eventz SET on_hold = 1 WHERE event_id = ?;",
		Postgres: "UPDATE eventz SET on_hold = 1 WHERE event_id = $1;",
	}.Switch(dbRW)
	if err != nil {
		return err
	}

	var result sql.Result
	result, err = dbRW.Exec(s, eventid)
	if err != nil {
		return err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row to be affected but %d were", rowsAffected)
	}
	return err
}

func resetSubscriberEvent(dbRW *sql.DB, eventid string) error {

	s, err := sqlbuilder.DriverSpecificSQL{
		MySQL:    "? AND on_hold = 1;",
		Postgres: "$1 AND on_hold = 1;",
	}.Switch(dbRW)
	if err != nil {
		return err
	}
	s = "UPDATE eventz SET on_hold = 0 WHERE event_id = " + s

	var result sql.Result
	result, err = dbRW.Exec(s, eventid)
	if err != nil {
		return err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row to be affected but %d were", rowsAffected)
	}
	return err
}

func logError(logto io.Writer, err error) {
	var cause error
	if ec, ok := err.(ErrorWithCause); ok {
		cause = ec.Cause()
	}
	msg := fmt.Sprintf("[ERROR] %s", err.Error())
	if cause != nil {
		msg = msg + fmt.Sprintf(" [CAUSE] %v", cause)
	}
	fmt.Fprintln(logto, msg)
}
func logMetrics(logto io.Writer, key string, count int, duration float64) {
	msg := fmt.Sprintf("[METRICS] "+`{"key": "%s", "count": %d, "duration": %f}`, key, count, duration)
	fmt.Fprintln(logto, msg)
}
func logInfo(logto io.Writer, msg string) {
	fmt.Fprintln(logto, fmt.Sprintf("[INFO] %s", msg))
}
