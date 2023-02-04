package main

func initSchema() error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer dbcon.Release()
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `create schema bot`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		create table if not exists bot.file_ref (
			file_id varchar(128) primary key not null
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		create table if not exists bot.permissions (
			file_id varchar(128) not null,
			perm_id varchar(128) primary key not null,
			email varchar(128),
			role varchar(128) not null,
			role_type varchar(128) not null,
			constraint fk_file_ref
			foreign key (file_id)
				references bot.file_ref(file_id)
				on delete cascade
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
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
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		create table if not exists bot.members (
			member_id varchar(128) primary key not null,
			member_name varchar(128) not null
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		create table if not exists bot.role_ref (
			role_id varchar(128) primary key not null
		)
	`)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	return err
}

func isValidDatabase() (bool, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer dbcon.Release()
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
	err = row.Scan(&schemaExists)
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

func getMembersFromDB() ([]*Member, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer dbcon.Release()
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
