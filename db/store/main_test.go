package store

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testStore Store
	testDB    *pgxpool.Pool
)

const (
	dbSource = "postgresql://root:illchangeitlater@localhost:5433/bank-app?sslmode=disable"
)

func TestMain(m *testing.M) {
	var err error

	testDB, err = pgxpool.New(context.Background(), dbSource)
	// Ping the database, then:
	testStore = NewStore(testDB)

	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}

	err = testDB.Ping(context.Background())
	if err != nil {
		log.Fatal("cannot ping database:", err)
	}

	testStore = NewStore(testDB)

	os.Exit(m.Run())
}
