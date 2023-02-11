package main

import (
	"strconv"
	"time"

	"github.com/disgoorg/log"
	"google.golang.org/api/sheets/v4"
)

type SheetBatchUpdate struct {
	ID    string
	Batch *sheets.BatchUpdateSpreadsheetRequest
}

func RetrySheetBatchUpdate(req *SheetBatchUpdate, prevLimit, maxWaitSeconds float64, hasSuggestedRetryDur bool) {
	var waitDur float64
	if hasSuggestedRetryDur {
		waitDur = prevLimit
	} else {
		waitDur = CalcThrottledWaitDuration(prevLimit, maxWaitSeconds)
	}
	<-time.After(time.Duration(waitDur) * time.Second)
	resp, err := gsheetsSvc.Spreadsheets.BatchUpdate(req.ID, req.Batch).Do()
	log.Debugf("google sheets batch update response HTTP status code: %d", resp.HTTPStatusCode)
	if resp.HTTPStatusCode == 429 {
		durStr := resp.Header.Get("Retry-After")
		var initWait float64
		innerHasSuggestedRetryDur := false
		if durStr == "" {
			initWait = waitDur
		} else {
			initWait, err = strconv.ParseFloat(durStr, 64)
			if err != nil {
				log.Error(err)
				return
			}
			innerHasSuggestedRetryDur = true
		}
		RetrySheetBatchUpdate(req, initWait, maxWaitSeconds, innerHasSuggestedRetryDur)
	}
	if err != nil {
		log.Error(err)
	}
}

func googleSheetBatchUpdateRateLimiter(rateLimit, maxRetryDuration float64, reqs <-chan *SheetBatchUpdate) {
	for {
		waitDur := CalcWaitDuration(rateLimit)
		req := <-reqs
		resp, err := gsheetsSvc.Spreadsheets.BatchUpdate(req.ID, req.Batch).Do()
		if resp != nil {
			log.Debugf("google sheets batch update response HTTP status code: %d", resp.HTTPStatusCode)
			if resp.HTTPStatusCode == 429 {
				durStr := resp.Header.Get("Retry-After")
				var initWait float64
				hasSuggestedRetryDur := false
				if durStr == "" {
					initWait = waitDur
				} else {
					initWait, err = strconv.ParseFloat(durStr, 64)
					if err != nil {
						log.Error(err)
						continue
					}
					hasSuggestedRetryDur = true
				}
				RetrySheetBatchUpdate(req, initWait, 3600, hasSuggestedRetryDur)
			}
		}
		if err != nil {
			log.Error(err)
		}
		log.Debugf("wrote google sheets batch with %d updates for spreadsheet %s", len(req.Batch.Requests), req.ID)
		<-time.After(time.Duration(waitDur) * time.Second)
	}
}
