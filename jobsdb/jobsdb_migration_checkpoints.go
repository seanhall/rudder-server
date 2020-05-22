package jobsdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-server/utils/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

//MigrationCheckpointT captures an event of export/import to recover from incase of a crash during migration
type MigrationCheckpointT struct {
	ID            int64           `json:"ID"`
	MigrationType MigrationOp     `json:"MigrationType"` //ENUM : export, import, acceptNewEvents
	FromNode      string          `json:"FromNode"`
	ToNode        string          `json:"ToNode"`
	FileLocation  string          `json:"FileLocation"`
	Status        string          `json:"Status"` //ENUM : Look up 'Values for Status'
	StartSeq      int64           `json:"StartSeq"`
	Payload       json.RawMessage `json:"Payload"`
	TimeStamp     time.Time       `json:"TimeStamp"`
}

//MigrationOp is a custom type for supported types in migrationCheckpoint
type MigrationOp string

//ENUM Values for MigrationOp
const (
	ExportOp          = "export"
	ImportOp          = "import"
	AcceptNewEventsOp = "acceptNewEvents"
)

//ENUM Values for Status
const (
	SetupForExport = "setup_for_export"
	Exported       = "exported"
	Notified       = "notified"
	Completed      = "completed"

	SetupToAcceptNewEvents = "setup_to_accept_new_events"
	SetupForImport         = "setup_for_import"
	PreparedForImport      = "prepared_for_import"
	Imported               = "imported"
)

//MigrationCheckpointSuffix : Suffix for checkpoints table
const (
	MigrationCheckpointSuffix = "migration_checkpoints"
	UniqueConstraintSuffix    = "unique_checkpoint"
)

//Checkpoint writes a migration event if id is passed as 0. Else it will update status and start_sequence
func (jd *HandleT) Checkpoint(migrationCheckpoint *MigrationCheckpointT) int64 {
	return jd.CheckpointInTxn(nil, migrationCheckpoint)
}

func (jd *HandleT) deleteCheckpoint(migrationCheckpoint *MigrationCheckpointT) {
	sqlStatement := fmt.Sprintf(`DELETE FROM %s WHERE id = %d`, jd.getCheckpointTableName(), migrationCheckpoint.ID)
	stmt, err := jd.dbHandle.Prepare(sqlStatement)
	_, err = stmt.Exec()
	jd.assertError(err)
}

//CheckpointInTxn writes a migration event if id is passed as 0. Else it will update status and start_sequence
// If txn is passed, it will run the statement in that txn, otherwise it will execute without a transaction
func (jd *HandleT) CheckpointInTxn(txn *sql.Tx, migrationCheckpoint *MigrationCheckpointT) int64 {
	jd.assert(migrationCheckpoint.MigrationType == ExportOp ||
		migrationCheckpoint.MigrationType == ImportOp ||
		migrationCheckpoint.MigrationType == AcceptNewEventsOp,
		fmt.Sprintf("MigrationType: %s is not a supported operation. Should be %s or %s",
			migrationCheckpoint.MigrationType, ExportOp, ImportOp))

	var sqlStatement string
	var checkpointType string
	if migrationCheckpoint.ID > 0 {
		sqlStatement = fmt.Sprintf(`UPDATE %s SET status = $1, start_sequence = $2, payload = $3 WHERE id = $4 RETURNING id`, jd.getCheckpointTableName())
		checkpointType = "update"
	} else {
		sqlStatement = fmt.Sprintf(`INSERT INTO %s (migration_type, from_node, to_node, file_location, status, start_sequence, payload, time_stamp)
									VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT ON CONSTRAINT %s DO UPDATE SET status=EXCLUDED.status RETURNING id`, jd.getCheckpointTableName(), jd.getUniqueConstraintName())
		checkpointType = "insert"
	}

	var (
		stmt *sql.Stmt
		err  error
	)
	logger.Infof("Checkpointing with query : %s with migrationCheckpoint %+v", sqlStatement, migrationCheckpoint)
	if txn != nil {
		stmt, err = txn.Prepare(sqlStatement)
	} else {
		stmt, err = jd.dbHandle.Prepare(sqlStatement)
	}
	jd.assertError(err)
	defer stmt.Close()

	var meID int64
	if migrationCheckpoint.ID > 0 {
		err = stmt.QueryRow(migrationCheckpoint.Status, migrationCheckpoint.StartSeq, migrationCheckpoint.Payload, migrationCheckpoint.ID).Scan(&meID)
	} else {
		err = stmt.QueryRow(migrationCheckpoint.MigrationType,
			migrationCheckpoint.FromNode,
			migrationCheckpoint.ToNode,
			migrationCheckpoint.FileLocation,
			migrationCheckpoint.Status,
			migrationCheckpoint.StartSeq,
			migrationCheckpoint.Payload,
			time.Now()).Scan(&meID)
	}
	if txn == nil {
		jd.assertError(err)
	}
	logger.Infof("%s-Migration: %s checkpoint %s from %s to %s. file: %s, status: %s for checkpointId: %d",
		jd.tablePrefix,
		migrationCheckpoint.MigrationType,
		checkpointType,
		migrationCheckpoint.FromNode,
		migrationCheckpoint.ToNode,
		migrationCheckpoint.FileLocation,
		migrationCheckpoint.Status,
		migrationCheckpoint.ID)
	return meID
}

//NewSetupCheckpointEvent returns a new migration event that captures setup for export, import of new event acceptance
func NewSetupCheckpointEvent(migrationType MigrationOp, node string) MigrationCheckpointT {
	switch migrationType {
	case ExportOp:
		return NewMigrationCheckpoint(migrationType, node, "All", "", SetupForExport, 0)
	case AcceptNewEventsOp:
		return NewMigrationCheckpoint(migrationType, "All", node, "", SetupToAcceptNewEvents, 0)
	case ImportOp:
		return NewMigrationCheckpoint(migrationType, "All", node, "", SetupForImport, 0)
	default:
		panic("Illegal usage")
	}
}

//NewMigrationCheckpoint is a constructor for MigrationCheckpoint struct
func NewMigrationCheckpoint(migrationType MigrationOp, fromNode string, toNode string, fileLocation string, status string, startSeq int64) MigrationCheckpointT {
	return MigrationCheckpointT{0, migrationType, fromNode, toNode, fileLocation, status, startSeq, []byte("{}"), time.Now()}
}

//SetupCheckpointTable creates a table
func (jd *HandleT) SetupCheckpointTable() {
	sqlStatement := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGSERIAL PRIMARY KEY,
		migration_type TEXT NOT NULL,
		from_node TEXT NOT NULL,
		to_node TEXT NOT NULL,
		file_location TEXT,
		status TEXT,
		start_sequence BIGINT,
		payload JSONB,
		time_stamp TIMESTAMP NOT NULL DEFAULT NOW(),
		CONSTRAINT %s UNIQUE(migration_type, from_node, to_node, file_location, status)
		);`, jd.getCheckpointTableName(), jd.getUniqueConstraintName())

	_, err := jd.dbHandle.Exec(sqlStatement)
	jd.assertError(err)
	logger.Infof("%s-Migration: %s table created", jd.GetTablePrefix(), jd.getCheckpointTableName())
}

func (jd *HandleT) getCheckpointTableName() string {
	return fmt.Sprintf("%s_%d_%d_%s", jd.GetTablePrefix(), misc.GetMigratingFromVersion(), misc.GetMigratingToVersion(), MigrationCheckpointSuffix)
}

func (jd *HandleT) getUniqueConstraintName() string {
	return fmt.Sprintf("%s_%d_%d_%s", jd.GetTablePrefix(), misc.GetMigratingFromVersion(), misc.GetMigratingToVersion(), UniqueConstraintSuffix)
}

func (jd *HandleT) findDsFromSetupCheckpoint(migrationType MigrationOp) (dataSetT, bool) {
	setupEvent := jd.GetSetupCheckpoint(migrationType)
	if setupEvent == nil {
		return dataSetT{}, false
	}
	ds := dataSetT{}
	err := json.Unmarshal(setupEvent.Payload, &ds)
	jd.assertError(err)
	return ds, true
}

func (jd *HandleT) createSetupCheckpointAndGetDs(migrationType MigrationOp) dataSetT {
	jd.dsListLock.Lock()
	defer jd.dsListLock.Unlock()

	dsList := jd.getDSList(true)
	me := NewSetupCheckpointEvent(migrationType, misc.GetNodeID())

	var ds dataSetT
	switch migrationType {
	case ExportOp:
		ds = jd.getLastDsForExport(dsList)
	case AcceptNewEventsOp:
		ds = jd.getDsForNewEvents(dsList)
	case ImportOp:
		//TODO If addNewDS is done and crashed, then goes into crashloop
		ds = jd.getDsForImport(dsList)
	}

	var err error
	me.Payload, err = json.Marshal(ds)
	if err != nil {
		panic("Unable to Marshal")
	}
	jd.Checkpoint(&me)
	return ds
}

func (jd *HandleT) findOrCreateDsFromSetupCheckpoint(migrationType MigrationOp) dataSetT {
	ds, found := jd.findDsFromSetupCheckpoint(migrationType)
	if !found {
		ds = jd.createSetupCheckpointAndGetDs(migrationType)
	}
	return ds
}

func (jd *HandleT) getSeqNoForFileFromDB(fileLocation string, migrationType MigrationOp) int64 {
	jd.assert(migrationType == ExportOp ||
		migrationType == ImportOp,
		fmt.Sprintf("MigrationType: %s is not a supported operation. Should be %s or %s",
			migrationType, ExportOp, ImportOp))

	sqlStatement := fmt.Sprintf(`SELECT start_sequence from %s WHERE file_location = $1 AND migration_type = $2 ORDER BY id DESC`, jd.getCheckpointTableName())
	stmt, err := jd.dbHandle.Prepare(sqlStatement)
	defer stmt.Close()
	jd.assertError(err)

	rows, err := stmt.Query(fileLocation, migrationType)
	defer rows.Close()
	if err != nil {
		panic("Unable to query")
	}
	rows.Next()

	var sequenceNumber int64
	sequenceNumber = 0
	err = rows.Scan(&sequenceNumber)
	if err != nil && err.Error() != "sql: Rows are closed" {
		panic("query result pares issue")
	}
	return sequenceNumber
}

//GetSetupCheckpoint gets all checkpoints and picks out the setup event for that type
func (jd *HandleT) GetSetupCheckpoint(migrationType MigrationOp) *MigrationCheckpointT {
	var setupStatus string
	switch migrationType {
	case ExportOp:
		setupStatus = SetupForExport
	case AcceptNewEventsOp:
		setupStatus = SetupToAcceptNewEvents
	case ImportOp:
		setupStatus = SetupForImport
	}
	setupEvents := jd.GetCheckpoints(migrationType, setupStatus)

	switch len(setupEvents) {
	case 0:
		return nil
	case 1:
		return setupEvents[0]
	default:
		panic("More than 1 setup event found. This should not happen")
	}

}

//GetCheckpoints gets all checkpoints and
func (jd *HandleT) GetCheckpoints(migrationType MigrationOp, status string) []*MigrationCheckpointT {
	sqlStatement := fmt.Sprintf(`SELECT * from %s WHERE migration_type = $1 AND status = $2 ORDER BY ID ASC`, jd.getCheckpointTableName())
	stmt, err := jd.dbHandle.Prepare(sqlStatement)
	jd.assertError(err)
	defer stmt.Close()

	rows, err := stmt.Query(migrationType, status)
	if err != nil {
		panic("Unable to query")
	}
	defer rows.Close()

	migrationCheckpoints := []*MigrationCheckpointT{}
	for rows.Next() {
		migrationCheckpoint := MigrationCheckpointT{}

		err = rows.Scan(&migrationCheckpoint.ID, &migrationCheckpoint.MigrationType, &migrationCheckpoint.FromNode,
			&migrationCheckpoint.ToNode, &migrationCheckpoint.FileLocation, &migrationCheckpoint.Status,
			&migrationCheckpoint.StartSeq, &migrationCheckpoint.Payload, &migrationCheckpoint.TimeStamp)
		if err != nil {
			panic(fmt.Sprintf("query result pares issue : %s", err.Error()))
		}
		migrationCheckpoints = append(migrationCheckpoints, &migrationCheckpoint)
	}
	return migrationCheckpoints
}

func getNumberOfJobsFromFileLocation(fileLocation string) int64 { //TODO: Don't get from file location. Explicitly store it
	slicedS := strings.FieldsFunc(fileLocation, fileLocationSplitter)
	totalJobs, _ := strconv.ParseInt(slicedS[len(slicedS)-2], 10, 64)
	return totalJobs
}

func fileLocationSplitter(r rune) bool {
	return r == '_' || r == '.'
}

func (migrationCheckpoint *MigrationCheckpointT) getLastJobID() int64 {
	if migrationCheckpoint.StartSeq == 0 {
		return int64(0)
	}
	return migrationCheckpoint.StartSeq + getNumberOfJobsFromFileLocation(migrationCheckpoint.FileLocation) - 1
}