package main

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initSchema(dbcon *pgxpool.Conn) []error {
	b := &pgx.Batch{}
	b.Queue(`create schema bot`)
	b.Queue(`
		create table if not exists bot.file_ref (
			file_id varchar(128) primary key not null
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
	b.Queue(`
		create table if not exists bot.members (
			member_id varchar(128) primary key not null,
			member_name varchar(128) not null
		)
	`)
	b.Queue(`
		create table if not exists bot.role_ref (
			role_id varchar(128) primary key not null
		)
	`)

	bresults := dbcon.SendBatch(ctx, b)
	errs := make([]error, b.Len())
	for i := 0; i < b.Len(); i++ {
		_, errs[i] = bresults.Exec()
	}
	return errs
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

type MemberID string
type Member struct {
	id   MemberID
	name string
}

func getMembersFromDB(dbcon *pgxpool.Conn) ([]*Member, error) {
	query := `
		select
			member_id,
			member_name
		from bot.members
		order by member_name
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}

	members := []*Member{}
	for rows.Next() {
		var memberID string
		var membername string
		err = rows.Scan(&memberID, &membername)
		if err != nil {
			return nil, err
		}
		members = append(members, &Member{
			id:   MemberID(memberID),
			name: membername,
		})
	}
	return members, nil
}
