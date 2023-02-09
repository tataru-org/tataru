package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/log"
	"github.com/google/uuid"
)

func xivapiCollectCharacterSearchResponses(requests []XivCharacterSearchRequest) ([]XivCharacterSearch, error) {
	responses := make([]XivCharacterSearch, len(requests))
	for i := 0; i < len(requests); i++ {
		var tokenMap XivApiTokenMap
		go func() {
			xivapiLodestoneReqs <- requests[i]
		}()
		// collect the token map
		maxIters := 1000
		iters := 0
		log.Debugf("waiting for token map for character %s", requests[i].Name)
		for {
			if iters == maxIters {
				return nil, fmt.Errorf("max iterations hit while waiting for xivapi token map")
			}
			// wait for token
			tMap := <-xivapiLodestoneReqTokens
			// check if this is the corresponding token map to the request that was sent
			if tMap.RequestToken != requests[i].Token {
				// send it back through the channel
				go func() {
					xivapiLodestoneReqTokens <- tMap
				}()
			} else {
				tokenMap = tMap
				break
			}
			iters++
		}
		iters = 0
		// collect the response
		log.Debugf("waiting for search response for character %s", requests[i].Name)
		for {
			if iters == maxIters {
				return nil, fmt.Errorf("max iterations hit while waiting for xivapi token map")
			}
			// wait for the response
			r := <-xivapiLodestoneResps
			// check if this is the corresponding response
			if respVal, ok := r[tokenMap]; ok {
				responses[i] = respVal.(XivCharacterSearch)
				break
			} else {
				// send it back through the channel
				go func() {
					xivapiLodestoneResps <- r
				}()
			}
			iters++
		}
	}
	return responses, nil
}

func xivapiCollectCharacterResponses(requests []XivCharacterRequest) ([]XivCharacter, error) {
	responses := make([]XivCharacter, len(requests))
	for i := 0; i < len(requests); i++ {
		var tokenMap XivApiTokenMap
		go func() {
			xivapiLodestoneReqs <- requests[i]
		}()
		// collect the token map
		maxIters := 1000
		iters := 0
		for {
			if iters == maxIters {
				return nil, fmt.Errorf("max iterations hit while waiting for xivapi token map")
			}
			// wait for token
			tMap := <-xivapiLodestoneReqTokens
			// check if this is the corresponding token map to the request that was sent
			if tMap.RequestToken != requests[i].Token {
				// send it back through the channel
				go func() {
					xivapiLodestoneReqTokens <- tMap
				}()
			} else {
				tokenMap = tMap
				break
			}
			iters++
		}
		iters = 0
		// collect the response
		for {
			if iters == maxIters {
				return nil, fmt.Errorf("max iterations hit while waiting for xivapi token map")
			}
			// wait for the response
			r := <-xivapiLodestoneResps
			// check if this is the corresponding response
			if respVal, ok := r[tokenMap]; ok {
				responses[i] = respVal.(XivCharacter)
				break
			} else {
				// send it back through the channel
				go func() {
					xivapiLodestoneResps <- r
				}()
			}
			iters++
		}
	}
	return responses, nil
}

func xivapiScanForCharacterIDs() {
	for {
		dbcon, err := dbpool.Acquire(ctx)
		if err != nil {
			log.Error(err)
			dbcon.Release()
			<-time.After(xivapiCharacterScanSleepDuration)
			continue
		}
		// get all members that have null xiv character IDs and create requests
		query := `
		select
			member_id,
			member_name
		from bot.members
		where member_xiv_id is null
		order by member_name
		`
		rows, err := dbcon.Query(
			ctx,
			query,
		)
		if err != nil {
			log.Error(err)
			dbcon.Release()
			<-time.After(xivapiCharacterScanSleepDuration)
			continue
		}
		reqMap := map[string]XivCharacterSearchRequest{}
		requests := []XivCharacterSearchRequest{}
		hasErr := false
		for rows.Next() {
			var memberID string
			var membername string
			err = rows.Scan(&memberID, &membername)
			if err != nil {
				hasErr = true
				break
			}
			req := XivCharacterSearchRequest{
				Token: uuid.New().String(),
				Name:  membername,
				Params: []XivApiQueryParam{
					{
						Name:  "server",
						Value: "Behemoth",
					},
				},
				Do: xivapiClient.SearchForCharacter,
			}
			reqMap[memberID] = req
			requests = append(requests, req)
		}
		log.Debug("created character name search requests")
		log.Debugf("# of character name search requests: %d", len(requests))
		if hasErr {
			log.Error(err)
			dbcon.Release()
			<-time.After(xivapiCharacterScanSleepDuration)
			continue
		}
		if len(requests) == 0 {
			dbcon.Release()
			<-time.After(xivapiCharacterScanSleepDuration)
			continue
		}
		// send requests and collect responses
		log.Debug("sending requests")
		responses, err := xivapiCollectCharacterSearchResponses(requests)
		log.Debug("responses collected")
		if err != nil {
			log.Error(err)
			dbcon.Release()
			<-time.After(xivapiCharacterScanSleepDuration)
			continue
		}
		// discord ID -> xiv character ID
		xivCharIDMap := map[string]string{}
		// determine if the character was found
		for i := 0; i < len(responses); i++ {
			characterProfiles := responses[i].Results
			for discordUserID, charSearchReq := range reqMap {
				for j := 0; j < len(characterProfiles); j++ {
					if characterProfiles[j].Name == charSearchReq.Name {
						xivCharIDMap[discordUserID] = strconv.FormatUint(uint64(characterProfiles[j].ID), 10)
						break
					}
				}
			}
		}
		log.Debugf("unpacked %d responses", len(xivCharIDMap))
		if len(xivCharIDMap) > 0 {
			// update the members where their xiv character ID was found
			tx, err := dbcon.Begin(ctx)
			if err != nil {
				log.Error(err)
				dbcon.Release()
				<-time.After(xivapiCharacterScanSleepDuration)
				continue
			}
			hasErr = false
			for discordUserID, xivCharacterID := range xivCharIDMap {
				_, err = tx.Exec(
					ctx,
					`update bot.members set member_xiv_id=$1 where member_id=$2`,
					xivCharacterID,
					discordUserID,
				)
				if err != nil {
					hasErr = true
					break
				}
			}
			if hasErr {
				log.Error(err)
				dbcon.Release()
				<-time.After(xivapiCharacterScanSleepDuration)
				continue
			}
			err = tx.Commit(ctx)
			if err != nil {
				log.Error(err)
			}
			log.Debugf("%d member character ids updated from auto-character ID search", len(xivCharIDMap))
		}
		log.Debug("auto-character ID seach successfully completed")
		dbcon.Release()
		<-time.After(xivapiCharacterScanSleepDuration)
	}
}
