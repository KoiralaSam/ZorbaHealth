package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitDB(ctx context.Context, dbURL string) error {

	config, err := pgxpool.ParseConfig(dbURL)

	if err != nil {
		return err
	}

	Pool, err = pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return err
	}

	err = Pool.Ping(ctx)

	if err != nil {
		return errors.New("failed to connect to database")
	}

	return nil

}

func GetDB() *pgxpool.Pool {
	return Pool
}
