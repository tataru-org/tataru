package main

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initSchema(dbcon *pgxpool.Conn) {
	b := &pgx.Batch{}
	b.Queue(`create schema bot`)
	b.Queue(`
		create table if not exists bot.file_ref (
			file_id varchar(128) primary key not null
		)
	`)
	b.Queue(`
		create table if not exists bot.users (
			file_id varchar(128) not null,
			user_id varchar(128) primary key not null,
			user_name varchar(128) not null,
			constraint fk_file_ref
				foreign key (file_id)
					references bot.file_ref(file_id)
					on delete cascade
		)
	`)
	b.Queue(`
		create table if not exists bot.permissions (
			file_id varchar(128) not null,
			perm_id varchar(128) primary key not null,
			constraint fk_file_ref
			foreign key (file_id)
				references bot.file_ref(file_id)
				on delete cascade
		)
	`)
	b.Queue(`
		create table if not exists bot.sheets (
			file_id varchar(128) not null,
			sheet_id varchar(128) primary key not null,
			expansion smallint unique not null,
			constraint fk_file_ref
			foreign key (file_id)
				references bot.file_ref(file_id)
				on delete cascade
		)
	`)
	dbcon.SendBatch(ctx, b)
}

func isValidDatabase(dbcon *pgxpool.Conn) (bool, error) {
	row := dbcon.QueryRow(
		ctx,
		`
			select exists (
				select 1
				from pg_catalog.pg_namespace
				where nspname = 'bot'
			)
		`,
	)
	var schemaExists bool
	err := row.Scan(&schemaExists)
	if err != nil {
		return false, err
	}
	return schemaExists, nil
}
