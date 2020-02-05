package whiz

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/homemade/whiz/internal/models"
	"github.com/homemade/whiz/subscribers"
	"github.com/labstack/echo"
)

type BackgroundWorkerStatus struct {
	apiStats        *sync.Map
	Started         time.Time
	PollingInterval time.Duration
	LastPolled      time.Time
	LastRun         time.Time
}

type BackgroundWorkerMetrics struct {
	API map[string]interface{}
}

func (s *BackgroundWorkerStatus) APIRequestResponse(requestURL string, responseStatusCode int) {
	var count int
	key := fmt.Sprintf("%d", responseStatusCode)
	v, ok := s.apiStats.Load(key)
	if ok {
		count = v.(int)
	}
	count = count + 1
	s.apiStats.Store(key, count)
}

func (s BackgroundWorkerStatus) Metrics() BackgroundWorkerMetrics {
	// gather metrics
	apiMetrics := make(map[string]interface{})
	s.apiStats.Range(func(k interface{}, v interface{}) bool {
		apiMetrics[k.(string)] = v
		return true
	})
	return BackgroundWorkerMetrics{
		API: apiMetrics,
	}
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

type BackgroundWorker struct {
	dbRW            *sql.DB
	dbRO            *sql.DB
	reg             subscribers.Registry
	loggerOutput    io.Writer
	lockName        string
	pollingInterval time.Duration
	errorCeiling    time.Duration
	tick            *time.Ticker
	mu              *sync.Mutex
	active          bool
	status          *BackgroundWorkerStatus
	shutdown        bool
}

func NewBackgroundWorker(dbRW *sql.DB, reg subscribers.Registry, loggeroutput io.Writer, options ...func(*BackgroundWorker) error) (*BackgroundWorker, error) {
	lockName := "whiz-worker"          // default
	pollingInterval := 1 * time.Minute // default
	errorCeiling := 1 * time.Minute    // default
	w := BackgroundWorker{
		dbRW:            dbRW,
		dbRO:            dbRW, // default
		reg:             reg,
		loggerOutput:    loggeroutput,
		lockName:        lockName,
		pollingInterval: pollingInterval,
		errorCeiling:    errorCeiling,
		mu:              &sync.Mutex{},
		status: &BackgroundWorkerStatus{
			apiStats: &sync.Map{},
		},
	}
	for _, option := range options {
		if err := option(&w); err != nil {
			return nil, err

		}
	}
	return &w, nil
}

func LockName(s string) func(*BackgroundWorker) error {
	return func(w *BackgroundWorker) error {
		w.lockName = s
		return nil
	}
}

func PollingInterval(t time.Duration) func(*BackgroundWorker) error {
	return func(w *BackgroundWorker) error {
		w.pollingInterval = t
		return nil
	}
}

func ErrorCeiling(t time.Duration) func(*BackgroundWorker) error {
	return func(w *BackgroundWorker) error {
		w.errorCeiling = t
		return nil
	}
}

// option for providing a read replica
func ReadOnlyDatabase(dbRO *sql.DB) func(*BackgroundWorker) error {
	return func(w *BackgroundWorker) error {
		w.dbRO = dbRO
		return nil
	}
}

func (w *BackgroundWorker) pollingLoop(i int) {
	for t := range w.tick.C {
		w.status.LastPolled = t
		if w.active {
			logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker cancelled next poll as previous run still active %v", w.status))
			continue // handle runs that take longer than the event loop polling interval
		}
		w.status.LastRun = t
		logMetrics(w.loggerOutput, "worker_poll", i+1, time.Since(w.status.Started).Seconds())
		w.nextRun(t)
		// handle changes to polling interval whilst running
		if w.pollingInterval != w.status.PollingInterval {
			logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker restarting polling as interval has changed from %v to %v", w.status.PollingInterval, w.pollingInterval))
			w.tick.Stop()
			w.tick = time.NewTicker(w.pollingInterval)
			w.status.PollingInterval = w.pollingInterval
			w.pollingLoop(i)
		}
	}
}

func (w *BackgroundWorker) StartPolling() {
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("BackgroundWorker returned from call to start polling with error %v", r))
		}
	}()
	w.status.Started = time.Now()
	w.status.PollingInterval = w.pollingInterval
	w.tick = time.NewTicker(w.pollingInterval)
	go func() {
		i := 0
		w.pollingLoop(i)
	}()
	logInfo(w.loggerOutput, "BackgroundWorker started polling")

}

func (w *BackgroundWorker) PollNow() {
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("BackgroundWorker returned from call to poll now with error %v", r))
		}
	}()
	t := time.Now()
	w.status.LastPolled = t
	if w.active {
		logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker cancelled poll now request as previous run still active %v", w.status))
		return // already polling
	}
	w.status.LastRun = t
	logMetrics(w.loggerOutput, "worker_poll_now", 1, time.Since(t).Seconds())
	w.nextRun(t)
}

func (w BackgroundWorker) Status() BackgroundWorkerStatus {
	return *w.status
}

func (w *BackgroundWorker) nextRun(t time.Time) {
	// ensure only one background worker can run at a time using an in memory check
	// we also use a database lock in case we accidently have multiple instances deployed see runEventLoops()
	w.mu.Lock()
	w.active = true
	w.mu.Unlock()
	defer func(wrk *BackgroundWorker) {
		wrk.mu.Lock()
		wrk.active = false
		wrk.mu.Unlock()
	}(w)
	w.runEventLoops(t)
}

func (w *BackgroundWorker) Shutdown() {
	defer func() {
		if r := recover(); r != nil {
			logError(w.loggerOutput, fmt.Errorf("BackgroundWorker returned from call to shutdown with error %v", r))
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

func (w *BackgroundWorker) runEventLoops(t time.Time) {

	// NOTE: use read only db where at all possible in the worker (this frees up the read write db for the web server)

	// we also need to handle long running processes again to ensure only 1 worker is ever running
	// we utilise MySQL locks for this
	var err error
	var lockTran *sql.Tx // we need to keep a handle on the transaction that opens the lock so we can release it later
	lockTran, err = w.dbRO.Begin()
	var locked int
	if err = lockTran.QueryRow(fmt.Sprintf("SELECT GET_LOCK('%s',1);", w.lockName)).Scan(&locked); err != nil {
		logError(w.loggerOutput, &echo.HTTPError{Message: "failed to acquire worker lock", Internal: err})
		return
	}
	if locked != 1 { // failed to acquire worker lock - another worker process is still running so just return
		logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker cancelled next run as detected another worker process running %v", w.status))
		return
	}
	defer func() {
		var unlocked int
		if err = lockTran.QueryRow(fmt.Sprintf("SELECT RELEASE_LOCK('%s');", w.lockName)).Scan(&unlocked); err != nil {
			logError(w.loggerOutput, &echo.HTTPError{Message: "failed to release worker lock during clean exit", Internal: err})
		} else {
			if unlocked != 1 {
				logError(w.loggerOutput, &echo.HTTPError{Message: "failed to release worker lock during clean exit"})
			}
		}
		if err = lockTran.Commit(); err != nil {
			logError(w.loggerOutput, &echo.HTTPError{Message: "failed to commit worker lock during clean exit", Internal: err})
		}
		return
	}()

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
					logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker subscriber group %d done with error %v", grp, err))
				} else {
					if r := recover(); r != nil {
						logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker subscriber group %d done with unhandled error %v", grp, r))
					} else {
						logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker subscriber group %d done", grp))
					}
				}
			}()
			// NOTE: use read only db where at all possible in the worker (this frees up the read/write db for the web server)
			var dbRoutines []models.EventzSubscriber
			dbRoutines, err = readRoutinesFromDB(w.dbRO, grp, w.errorCeiling)
			if err != nil {
				logError(w.loggerOutput, &echo.HTTPError{Message: "failed to read dbRoutines", Code: http.StatusInternalServerError, Internal: err})
				return
			}
			logMetrics(w.loggerOutput, fmt.Sprintf("worker_routines:%d", grp), len(dbRoutines), time.Since(t).Seconds())
			for _, r := range dbRoutines {
				var sr subscribers.Routine
				sr, err = w.reg.LoadRoutine(r.Name, r.Version, r.Instance, r.MetaData, w.status)
				if err != nil {
					logError(w.loggerOutput, &echo.HTTPError{Message: "failed to load subscriber routine", Code: http.StatusInternalServerError, Internal: err})
					return
				}
				err = updateLastRunAtInDB(w.dbRW, r)
				if err != nil {
					logError(w.loggerOutput, &echo.HTTPError{Message: fmt.Sprintf("failed to update last_run_at during execution of %s", r.Desc()), Code: http.StatusInternalServerError, Internal: err})
					return
				}
				var events SubscriberEvents
				events, err = readSubscriberEventsFromDB(w.dbRO, r.Group, r.Priority, sr.EventSources(), sr.EventLoopCeiling())
				if err != nil {
					logError(w.loggerOutput, &echo.HTTPError{Message: fmt.Sprintf("failed to read subscriber events during execution of %s", r.Desc()), Code: http.StatusInternalServerError, Internal: err})
					return
				}
				logMetrics(w.loggerOutput, fmt.Sprintf("worker_events:%d.%d", r.Group, r.Priority), len(events), time.Since(t).Seconds())
				processedEvents := 0
				for _, e := range events {
					// handle shutdown requests
					if w.shutdown {
						logInfo(w.loggerOutput, fmt.Sprintf("BackgroundWorker subscriber group %d exiting event loop due to shutdown request", grp))
						break
					}
					var procResult subscribers.Result
					procResult, err = sr.ProcessEvent(e)
					if err != nil {
						temporaryError, status := sr.IsTemporaryError(err)
						if !temporaryError { // we don't put events on hold for temporary errors - we will try again later
							if dberr := markSubscriberEventAsOnHold(w.dbRW, e.EventID); dberr != nil {
								logError(w.loggerOutput, &echo.HTTPError{Message: fmt.Sprintf("failed to mark event#%s as onhold", e.EventID), Code: http.StatusInternalServerError, Internal: dberr})
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
						cause := err
						if he, ok := err.(*echo.HTTPError); ok {
							cause = he.Internal
						}
						logError(w.loggerOutput, &echo.HTTPError{Message: msg, Code: status, Internal: cause})
						if err2 := updateLastErrorAtInDB(w.dbRW, r); err2 != nil {
							logError(w.loggerOutput, &echo.HTTPError{Message: fmt.Sprintf("failed to update last_error_at during execution of %s", r.Desc()), Code: http.StatusInternalServerError, Internal: err2})
						}
						// try and also add error to the log table
						insertIntoErrorLog(w.loggerOutput, w.dbRW, r, err, e.EventID, procResult)

						// dont break out of the loop for temporary 424 Failed Dependency errors
						// (these are a special case of temporary error used to try and handle events that are out of sequence)
						temporaryFailedDependencyError := temporaryError && (status == http.StatusFailedDependency)
						// and skippable errors - these will be left on hold
						if !temporaryFailedDependencyError || !sr.IsSkippableError(err) {
							break
						}

					} else {
						processedEvents = processedEvents + 1
						err = markSubscriberEventAsProcessed(w.dbRW, r, e, procResult)
						if err != nil {
							logError(w.loggerOutput, &echo.HTTPError{Message: fmt.Sprintf("failed to mark event#%s as processed", e.EventID), Code: http.StatusInternalServerError, Internal: err})
							return
						}
					}
				}
				logMetrics(w.loggerOutput, fmt.Sprintf("worker_processed:%d.%d", r.Group, r.Priority), processedEvents, time.Since(t).Seconds())
				if err != nil {
					return
				}
			}
		}(i)
	}
	// wait until all are complete
	wg.Wait()

	logMetrics(w.loggerOutput, "worker_execution", 1, time.Since(t).Seconds())
}

func readRoutinesFromDB(db *sql.DB, group uint, errceiling time.Duration) ([]models.EventzSubscriber, error) {
	routinesSQL := fmt.Sprintf("select name, version, instance, meta_data, priority from eventz_subscribers where `group` = %d and "+
		"priority > 0 and `group` in (select `group` from eventz_subscribers group by `group` "+
		"having MAX(last_error_at) is null or MAX(last_error_at) < DATE_SUB(NOW(6),INTERVAL %f SECOND)) "+
		"order by priority asc", group, errceiling.Seconds())
	fmt.Println(routinesSQL)
	result := make([]models.EventzSubscriber, 0)
	var (
		name      string
		version   string
		instance  string
		meta_data sql.NullString
		priority  uint
	)
	rows, err := db.Query(routinesSQL)
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
	result, err := dbRW.Exec("update eventz_subscribers set last_run_at = NOW(6) where name = ? and version = ? and instance = ?", sub.Name, sub.Version, sub.Instance)
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
	result, err := dbRW.Exec("update eventz_subscribers set last_error_at = NOW(6) where name = ? and version = ? and instance = ?", sub.Name, sub.Version, sub.Instance)
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

func insertIntoErrorLog(loggerOutput io.Writer, dbRW *sql.DB, sub models.EventzSubscriber, e error, eventid string, res subscribers.Result) {

	stmt, err := dbRW.Prepare(`INSERT INTO eventz_subscriber_error_logs
(event_id,routine_name,routine_version,routine_instance,error_code,error_message,status,refer_table,refer_id,created_at)
VALUES(?,?,?,?,?,?,?,?,?,NOW(6));`)
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
	var code int
	var cause error
	if he, ok := e.(*echo.HTTPError); ok {
		code = he.Code
		cause = he.Internal
	}
	_, err = stmt.Exec(eventid, sub.Name, sub.Version, sub.Instance, code, fmt.Sprintf("Error: %v \nCause: %v", e, cause), res.Status(), res.ReferTable(), res.ReferID())
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

	subscriberEventsSQL := fmt.Sprintf(`select event_id, event_created_at, event_source, event_uuid, model, type, action, user_id, model_data, source_data
from eventz
where on_hold = 0
and sub_grp%d_status = (%d - 1)`, group, priority)
	// sources can be a wild card
	if sources[0] == "*" {
		subscriberEventsSQL = subscriberEventsSQL + fmt.Sprintf(`
		and event_created_at < DATE_SUB(NOW(6),INTERVAL %d MINUTE)
		order by event_created_at,event_id asc`, ceiling)
	} else { // or a filter
		eventSources := strings.Join(strings.Fields(fmt.Sprint(sources)), "','")
		eventSources = strings.Replace(eventSources, "[", "('", 1)
		eventSources = strings.Replace(eventSources, "]", "')", 1)
		subscriberEventsSQL = subscriberEventsSQL + fmt.Sprintf(`
		and event_source in %s
		and event_created_at < DATE_SUB(NOW(6),INTERVAL %d MINUTE)
		order by event_created_at,event_id asc`, eventSources, ceiling)
	}
	rows, err := db.Query(subscriberEventsSQL)
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
	stmt := fmt.Sprintf("update eventz set sub_grp%d_status = ? where event_id = ?;", sub.Group)
	result, err := dbRW.Exec(stmt, sub.Priority, event.EventID)
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
		stmt = `INSERT INTO eventz_subscriber_processed_logs
		(event_id,routine_name,routine_version,routine_instance,meta_data,status,refer_table,refer_id,created_at)
		VALUES(?,?,?,?,?,?,?,?,NOW(6));`

		result, err = dbRW.Exec(stmt, event.EventID, sub.Name, sub.Version, sub.Instance, res.MetaData(), res.Status(), res.ReferTable(), res.ReferID())
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
	stmt := "update eventz set on_hold = 1 where event_id = ?;"
	result, err := dbRW.Exec(stmt, eventid)
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
	var status int
	var cause error
	if he, ok := err.(*echo.HTTPError); ok {
		status = he.Code
		cause = he.Internal
	}
	msg := fmt.Sprintf("[ERROR] %d %s", status, err.Error())
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
