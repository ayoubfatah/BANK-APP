package main

import (
	"context"
	"log"

	"github.com/bank-app/api"
	"github.com/bank-app/db/store"
	"github.com/bank-app/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := utils.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, config.DBURL)
	if err != nil {
		log.Fatal("cannot connect to db: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("cannot ping db: ", err)
	}

	st := store.NewStore(pool)
	server , err  := api.NewServer(config,st)

	if  err != nil {
		log.Fatal("cannot create server: ", err)
	}
	if err := server.Start(config.ServerAddress); err != nil {
		log.Fatal("cannot start server: ", err)
	}
}