package main

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type InitDataPath string

const (
	InitDataRootPath              InitDataPath = "/app/initial-db-data"
	InitDataBossExpansionMapPath  InitDataPath = InitDataRootPath + "/bot.boss_expansion_map.csv"
	InitDataBossMetadataPath      InitDataPath = InitDataRootPath + "/bot.boss_metadata.csv"
	InitDataBossMountMapPath      InitDataPath = InitDataRootPath + "/bot.boss_mount_map.csv"
	InitDataBossStylingDataPath   InitDataPath = InitDataRootPath + "/bot.boss_styling_data.csv"
	InitDataExpansionMetadataPath InitDataPath = InitDataRootPath + "/bot.expansion_metadata.csv"
	InitDataMountMetadataPath     InitDataPath = InitDataRootPath + "/bot.mount_metadata.csv"
)

func getInitDataPaths() []InitDataPath {
	return []InitDataPath{
		InitDataBossExpansionMapPath,
		InitDataBossMetadataPath,
		InitDataBossMountMapPath,
		InitDataBossStylingDataPath,
		InitDataExpansionMetadataPath,
		InitDataMountMetadataPath,
	}
}

type InitDataObjectName string

const (
	InitDataSchemaName                 InitDataObjectName = "bot"
	InitDataBossExpansionMapTableName  InitDataObjectName = "boss_expansion_map"
	InitDataBossMetadataTableName      InitDataObjectName = "boss_metadata"
	InitDataBossMountMapTableName      InitDataObjectName = "boss_mount_map"
	InitDataBossStylingDataTableName   InitDataObjectName = "boss_styling_data"
	InitDataExpansionMetadataTableName InitDataObjectName = "expansion_metadata"
	InitDataMountMetadataTableName     InitDataObjectName = "mount_metadata"
)

func getInitDataTableMap() map[InitDataPath]InitDataObjectName {
	return map[InitDataPath]InitDataObjectName{
		InitDataBossExpansionMapPath:  InitDataBossExpansionMapTableName,
		InitDataBossMetadataPath:      InitDataBossMetadataTableName,
		InitDataBossMountMapPath:      InitDataBossMountMapTableName,
		InitDataBossStylingDataPath:   InitDataBossStylingDataTableName,
		InitDataExpansionMetadataPath: InitDataExpansionMetadataTableName,
		InitDataMountMetadataPath:     InitDataMountMetadataTableName,
	}
}

func initDB() error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()

	// create the db objects
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return fmt.Errorf("dbcon.Begin() 1 error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `create schema bot`)
	if err != nil {
		return fmt.Errorf("create schema error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.file_ref (
			file_gcp_id varchar(128) primary key not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.file_ref error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.permissions (
			file_gcp_id varchar(128) not null,
			perm_gcp_id varchar(128) primary key not null,
			email varchar(128),
			role varchar(128) not null,
			role_type varchar(128) not null,
			constraint fk_file_ref
				foreign key (file_gcp_id)
					references bot.file_ref(file_gcp_id)
					on delete cascade
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.permissions error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.expansion_metadata (
			expansion_id varchar(36) primary key not null,
			expansion_name varchar(128) not null,
			expansion_index int not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.expansion_metadata error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.sheet_metadata (
			file_gcp_id varchar(128) not null,
			sheet_gcp_id varchar(128) primary key not null,
			sheet_index int not null,
			constraint fk_file_ref
				foreign key (file_gcp_id)
					references bot.file_ref(file_gcp_id)
					on delete cascade
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.sheet_metadata error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.sheet_expansion_map (
			sheet_gcp_id varchar(128) not null,
			expansion_id varchar(36) not null,
			primary key (
				sheet_gcp_id,
				expansion_id
			),
			constraint fk_sheet_gcp_id
				foreign key (sheet_gcp_id)
					references bot.sheet_metadata(sheet_gcp_id)
					on delete cascade
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.sheet_expansion_map error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.member_metadata (
			member_discord_id varchar(128) primary key not null,
			member_name varchar(128) not null,
			member_xiv_id varchar(128)
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.member_metadata error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.role_ref (
			role_id varchar(128) primary key not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.role_ref error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.boss_metadata (
			boss_id varchar(36) primary key not null,
			boss_name varchar(128) not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.boss_metadata error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.boss_expansion_map (
			boss_id varchar(36) not null,
			expansion_id varchar(36) not null,
			boss_expansion_index int not null,
			primary key (
				boss_id,
				expansion_id
			)
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.boss_expansion_map error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.mount_metadata (
			mount_id varchar(36) primary key not null,
			mount_name varchar(128) not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.mount_metadata error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.boss_mount_map (
			boss_id varchar(36) not null,
			mount_id varchar(36) not null,
			primary key (
				boss_id,
				mount_id
			)
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.boss_mount_map error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.member_data (
			member_discord_id varchar(128) not null,
			mount_id varchar(36) not null,
			has_mount boolean not null,
			primary key (
				member_discord_id,
				mount_id
			),
			constraint fk_member_discord_id
				foreign key (member_discord_id)
					references bot.member_metadata(member_discord_id)
					on delete cascade
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.member_data error: [%w]", err)
	}
	_, err = tx.Exec(ctx, `
		create table bot.boss_styling_data (
			boss_id varchar(36) primary key not null,
			header_background_hex_color varchar(9) not null,
			header_foreground_hex_color varchar(9) not null,
			checkbox_background_hex_color varchar(9) not null,
			checkbox_foreground_hex_color varchar(9) not null
		)
	`)
	if err != nil {
		return fmt.Errorf("create bot.boss_styling_data error: [%w]", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit() 1 error: [%w]", err)
	}

	// insert initial data into db
	initPaths := getInitDataPaths()
	for i := 0; i < len(initPaths); i++ {
		initPath := initPaths[i]
		file, err := os.Open(string(initPath))
		if err != nil {
			panic(err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		headerRow, err := reader.Read()
		if err != nil {
			panic(err)
		}
		data, err := reader.ReadAll()
		if err != nil {
			panic(err)
		}

		csvData := make([][]interface{}, len(data))
		for i := 0; i < len(data); i++ {
			row := data[i]
			csvRow := make([]interface{}, len(row))
			for j := 0; j < len(row); j++ {
				csvRow[j] = row[j]
			}
			csvData[i] = csvRow
		}

		initMap := getInitDataTableMap()
		_, err = dbcon.CopyFrom(
			ctx,
			pgx.Identifier(
				[]string{
					string(InitDataSchemaName),
					string(initMap[initPath]),
				},
			),
			headerRow,
			pgx.CopyFromRows(csvData),
		)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func isValidDatabase() (bool, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("database connection acquire error: [%w]", err)
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
		return false, fmt.Errorf("row scan error: [%w]", err)
	}
	return schemaExists, nil
}

type MemberID string
type Member struct {
	id    MemberID
	name  string
	xivid *string
}

func getMembersFromDB() ([]*Member, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select
			member_discord_id,
			member_name,
			member_xiv_id
		from bot.member_metadata
		order by member_name
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("get bot.member_metadata error: [%w]", err)
	}

	members := []*Member{}
	for rows.Next() {
		var memberID string
		var membername string
		var xivid *string
		err = rows.Scan(&memberID, &membername, &xivid)
		if err != nil {
			return nil, fmt.Errorf("row scan error: [%w]", err)
		}
		members = append(members, &Member{
			id:    MemberID(memberID),
			name:  membername,
			xivid: xivid,
		})
	}
	return members, nil
}
