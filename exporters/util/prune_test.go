package util

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/algorand/go-codec/codec"
	"github.com/algorand/indexer/idb"
	"github.com/algorand/indexer/idb/postgres"
	pgtest "github.com/algorand/indexer/idb/postgres/testing"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

var logger *logrus.Logger
var hook *test.Hook
var jsonCodecHandle *codec.JsonHandle

// encodeJSON converts an object into JSON
func encodeJSON(obj interface{}) []byte {
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, jsonCodecHandle)
	enc.MustEncode(obj)
	return buf
}
func init() {
	logger, hook = test.NewNullLogger()
	jsonCodecHandle = new(codec.JsonHandle)
	//jsonCodecHandle.ErrorIfNoField = true
	//jsonCodecHandle.ErrorIfNoArrayExpand = true
	//jsonCodecHandle.Canonical = true
	//jsonCodecHandle.RecursiveEmptyCheck = true
	//jsonCodecHandle.HTMLCharsAsIs = true
	//jsonCodecHandle.Indent = 0
	//jsonCodecHandle.MapKeyAsString = true
}

var config = PruneConfigurations{
	Frequency: "once",
	Rounds:    10,
	Timeout:   10,
}

type ImportState struct {
	NextRoundToAccount uint64 `codec:"next_account_round"`
}

func TestDeleteEmptyTxnTable(t *testing.T) {
	db, connStr, shutdownFunc := pgtest.SetupPostgres(t)
	defer shutdownFunc()

	// init the tables
	idb, _, err := postgres.OpenPostgres(connStr, idb.IndexerDbOptions{}, nil)
	assert.NoError(t, err)
	defer idb.Close()

	// empty txn table
	err = populateTxnTable(db, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowsInTxnTable(db))

	// data manager
	dm := MakeDataManager(context.Background(), &config, idb, logger)
	var wg sync.WaitGroup
	wg.Add(1)
	go dm.Delete(&wg)
	wg.Wait()
	assert.Equal(t, 0, rowsInTxnTable(db))
}

func TestDeleteTxns(t *testing.T) {
	db, connStr, shutdownFunc := pgtest.SetupPostgres(t)
	defer shutdownFunc()

	// init the tables
	idb, _, err := postgres.OpenPostgres(connStr, idb.IndexerDbOptions{}, nil)
	assert.NoError(t, err)
	defer idb.Close()

	// add 20 records to txn table
	err = populateTxnTable(db, 20)
	assert.NoError(t, err)
	assert.Equal(t, 20, rowsInTxnTable(db))

	// data manager
	dm := MakeDataManager(context.Background(), &config, idb, logger)
	var wg sync.WaitGroup
	wg.Add(1)
	go dm.Delete(&wg)
	wg.Wait()
	// 10 rounds removed
	assert.Equal(t, 10, rowsInTxnTable(db))
	// check remaining rounds are correct
	assert.True(t, validateTxnTable(db, 11, 20))
}

func TestDeleteConfigs(t *testing.T) {
	db, connStr, shutdownFunc := pgtest.SetupPostgres(t)
	defer shutdownFunc()

	// init the tables
	idb, _, err := postgres.OpenPostgres(connStr, idb.IndexerDbOptions{}, nil)
	assert.NoError(t, err)
	defer idb.Close()

	// add 3 record to txn table
	err = populateTxnTable(db, 3)
	assert.NoError(t, err)
	assert.Equal(t, 3, rowsInTxnTable(db))

	// config.Rounds > rounds in DB
	config = PruneConfigurations{
		Frequency: "once",
		Rounds:    5,
	}
	dm := MakeDataManager(context.Background(), &config, idb, logger)
	assert.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go dm.Delete(&wg)
	wg.Wait()
	// delete didn't happen
	assert.Equal(t, 3, rowsInTxnTable(db))

	// config.Rounds == rounds in DB
	config.Rounds = 3
	dm = MakeDataManager(context.Background(), &config, idb, logger)
	assert.NoError(t, err)
	wg.Add(1)
	go dm.Delete(&wg)
	wg.Wait()
	// delete didn't happen
	assert.Equal(t, 3, rowsInTxnTable(db))

	// unsupported frequency
	config = PruneConfigurations{
		Frequency: "hourly",
		Rounds:    1,
	}
	dm = MakeDataManager(context.Background(), &config, idb, logger)
	assert.NoError(t, err)
	wg.Add(1)
	go dm.Delete(&wg)
	wg.Wait()
	// delete didn't happen
	assert.Equal(t, 3, rowsInTxnTable(db))
}

func TestDeleteDaily(t *testing.T) {
	db, connStr, shutdownFunc := pgtest.SetupPostgres(t)
	defer shutdownFunc()

	// init the tables
	idb, _, err := postgres.OpenPostgres(connStr, idb.IndexerDbOptions{}, nil)
	assert.NoError(t, err)
	defer idb.Close()

	// add 20 record to txn table
	err = populateTxnTable(db, 20)
	assert.NoError(t, err)
	assert.Equal(t, 20, rowsInTxnTable(db))

	config = PruneConfigurations{
		Frequency: "daily",
		Rounds:    15,
		Timeout:   5,
	}
	ctx, cancel := context.WithCancel(context.Background())
	dm := MakeDataManager(ctx, &config, idb, logger)
	var wg sync.WaitGroup
	wg.Add(1)
	go dm.Delete(&wg)
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()
	wg.Wait()
	assert.Equal(t, 15, rowsInTxnTable(db))

	//	reconnect
	config = PruneConfigurations{
		Frequency: "daily",
		Rounds:    10,
		Timeout:   5,
	}
	ctx, cancel = context.WithCancel(context.Background())
	dm = MakeDataManager(ctx, &config, idb, logger)
	wg.Add(1)
	go dm.Delete(&wg)
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()
	wg.Wait()
	assert.Equal(t, 10, rowsInTxnTable(db))
}

func populateTxnTable(db *pgxpool.Pool, n int) error {
	batch := &pgx.Batch{}
	// txn round starts at 1 because genesis block is empty
	for i := 1; i <= n; i++ {
		query := "INSERT INTO txn(round, intra, typeenum,asset,txn,extra) VALUES ($1,$2,$3,$4,$5,$6)"
		batch.Queue(query, i, i, 1, 0, "{}", "{}")
	}
	results := db.SendBatch(context.Background(), batch)
	defer results.Close()
	for i := 0; i < batch.Len(); i++ {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("populateTxnTable() exec err: %w", err)
		}
	}

	// set state in metastate
	setMetastateUpsert := `INSERT INTO metastate (k, v) VALUES ($1, $2)`
	importstate := ImportState{
		NextRoundToAccount: uint64(n + 1),
	}
	_, err := db.Exec(context.Background(), setMetastateUpsert, "state", string(encodeJSON(importstate)))
	if err != nil {
		return fmt.Errorf("populateTxnTable(): %w", err)
	}
	return nil
}

func rowsInTxnTable(db *pgxpool.Pool) int {
	var rows int
	r, err := db.Query(context.Background(), "SELECT count(*) FROM txn")
	if err != nil {
		return 0
	}
	defer r.Close()
	for r.Next() {
		err = r.Scan(&rows)
		if err != nil {
			return 0
		}
	}
	return rows
}

func validateTxnTable(db *pgxpool.Pool, first, last uint64) bool {
	res, err := db.Query(context.Background(), "SELECT round FROM txn")
	if err != nil {
		return false
	}
	defer res.Close()
	var round uint64
	// expected round
	expected := first
	for res.Next() {
		err = res.Scan(&round)
		if err != nil || round != expected {
			return false
		}
		expected++
	}
	return expected-1 == last
}