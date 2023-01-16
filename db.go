package main

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func truncateTable(dbcon *pgxpool.Conn, table string) error {
	query := fmt.Sprintf("truncate %s", table)
	_, err := dbcon.Exec(
		ctx,
		query,
	)
	return err
}
