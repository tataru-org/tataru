package main

import (
	"time"

	"github.com/disgoorg/log"
	"google.golang.org/api/sheets/v4"
)

type SheetBatchUpdate struct {
	ID    string
	Batch *sheets.BatchUpdateSpreadsheetRequest
}

func RetrySheetBatchUpdate(req *SheetBatchUpdate, prevLimit, maxWaitSeconds float64) {
	waitDur := CalcThrottledWaitDuration(prevLimit, maxWaitSeconds)
	<-time.After(time.Duration(waitDur) * time.Second)
	resp, err := gsheetsSvc.Spreadsheets.BatchUpdate(req.ID, req.Batch).Do()
	if resp.HTTPStatusCode == 429 {
		RetrySheetBatchUpdate(req, waitDur, maxWaitSeconds)
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
		if resp.HTTPStatusCode == 429 {
			RetrySheetBatchUpdate(req, waitDur, 3600)
		}
		if err != nil {
			log.Error(err)
		}
		log.Debugf("wrote google sheets batch with %d updates for spreadsheet %s", len(req.Batch.Requests), req.ID)
		<-time.After(time.Duration(waitDur) * time.Second)
	}
}
