package main

import (
	"runtime/debug"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
)

func onGuildReady(event *events.GuildReady) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Conn().Close(ctx)
	isValidDb, err := isValidDatabase(dbcon)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if isValidDb {
		log.Debug("schema is valid")
	} else {
		log.Debug("schema is invalid")
		errs := initSchema(dbcon)
		for i := 0; i < len(errs); i++ {
			if errs[i] != nil {
				log.Fatal(err)
				log.Fatal(debug.Stack())
				return
			}
		}
		log.Debug("schema initialized")
	}

	// check if db has a record of the file
	var fileRefExists bool
	row := dbcon.QueryRow(
		ctx,
		"select exists(select file_id from bot.file_ref)",
	)
	err = row.Scan(&fileRefExists)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}

	if fileRefExists {
		// check if the file exists in google drive
		var fileID string
		row := dbcon.QueryRow(
			ctx,
			"select file_id from bot.file_ref",
		)
		err = row.Scan(&fileID)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		ok, err := fileExists(FileID(fileID))
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		if *ok {
			// check if the file needs to be updated
			members, err := event.Client().Rest().GetMembers(event.GuildID)
			if err != nil {
				log.Fatal(err)
				log.Fatal(debug.Stack())
				return
			}
			syncRoleMembers(FileID(fileID), members)
		} else {
			// create the file
			log.Debug("file exists in db on startup but does not exist in google drive")
			buildFile(true)
			log.Debug("file built")
		}
	} else {
		// create the file
		log.Debug("file does not exist in db on startup")
		buildFile(false)
		log.Debug("file built")
	}
}
