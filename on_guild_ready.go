package main

import (
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
)

func onGuildReady(event *events.GuildReady) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()
	isValidDb, err := isValidDatabase()
	if err != nil {
		log.Error(err)
		return
	}
	if isValidDb {
		log.Debug("schema is valid")
	} else {
		log.Debug("schema is invalid")
		err := initDB()
		if err != nil {
			log.Error(err)
			return
		}
		log.Debug("schema initialized")
	}

	// check if db has a record of the file
	var fileRefExists bool
	row := dbcon.QueryRow(
		ctx,
		"select exists(select file_gcp_id from bot.file_ref)",
	)
	err = row.Scan(&fileRefExists)
	if err != nil {
		log.Error(err)
		return
	}

	go googleSheetBatchUpdateRateLimiter(googleSheetsWriteRateLimit, maxRetryDuration, googleSheetsWriteReqs)

	var fileID *FileID
	if fileRefExists {
		var fileIDStr string
		// check if the file exists in google drive
		row := dbcon.QueryRow(
			ctx,
			"select file_gcp_id from bot.file_ref",
		)
		err = row.Scan(&fileIDStr)
		if err != nil {
			log.Error(err)
			return
		}
		exists, err := fileExists(FileID(fileIDStr))
		if err != nil {
			log.Error(err)
			return
		}
		dbcon.Release()
		if !*exists {
			// create the file
			log.Debug("file exists in db on startup but does not exist in google drive")
			fileID, err = buildFile(false)
			if err != nil {
				log.Error(err)
				return
			}
			log.Debug("file built")
		} else {
			fileID = (*FileID)(&fileIDStr)
		}
	} else {
		dbcon.Release()
		// create the file
		log.Debug("file does not exist in db on startup")
		fileID, err = buildFile(false)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debug("file built")
	}

	// check if the file needs to be updated
	members, err := event.Client().Rest().GetMembers(event.GuildID, guildMemberCountRequestLimit, nullSnowflake)
	if err != nil {
		log.Error(err)
		return
	}
	err = syncRoleMembers(*fileID, members)
	if err != nil {
		log.Error(err)
		return
	}
	err = discordNicknameScan(members)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("sync successfully completed")
	go xivApiLodestoneRequestRateLimiter(xivapiLodestoneRateLimit, maxRetryDuration, xivapiLodestoneReqs, xivapiLodestoneResps, xivapiLodestoneReqTokens)
	go xivapiScanForCharacterIDs()
	go scanForMounts()
	log.Debug("routines launched")
}
