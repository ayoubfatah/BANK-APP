package store

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbSource = "postgresql://root:illchangeitlater@localhost:5433/bank-app?sslmode=disable"
)

var testStore *Store

func TestMain(m *testing.M) {
	var err error

	testDb, err := pgxpool.New(context.Background(), dbSource)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}

	err = testDb.Ping(context.Background())
	if err != nil {
		log.Fatal("cannot ping database:", err)
	}

	testStore = NewStore(testDb)

	os.Exit(m.Run())
}