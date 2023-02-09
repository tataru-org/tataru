package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/disgoorg/log"
	"github.com/google/uuid"
)

type XivApiTokenMap struct {
	RequestToken  string
	ResponseToken string
}

func RetryXivApiLodestoneRequest(req interface{}, prevLimit, maxWaitSeconds float64, hasSuggestedRetryDur bool) (interface{}, error) {
	var waitDur float64
	if hasSuggestedRetryDur {
		waitDur = prevLimit
	} else {
		waitDur = CalcThrottledWaitDuration(prevLimit, maxWaitSeconds)
	}
	<-time.After(time.Duration(waitDur) * time.Second)
	var out interface{}
	switch r := req.(type) {
	case XivCharacterSearchRequest:
		resp, err := r.Do(r.Name, r.Params...)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		log.Debugf("retry xivapi lodestone character search request api reponse status code: %d", resp.StatusCode)
		if resp.StatusCode == 429 {
			durStr := resp.Header.Get("Retry-After")
			var initWait float64
			hasSuggestedRetryDur := false
			if durStr == "" {
				initWait = waitDur
			} else {
				initWait, err = strconv.ParseFloat(durStr, 64)
				if err != nil {
					log.Error(err)
					return nil, err
				}
				hasSuggestedRetryDur = true
			}
			return RetryXivApiLodestoneRequest(r, initWait, 3600, hasSuggestedRetryDur)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var characterSearch XivCharacterSearch
		err = json.Unmarshal(respBody, &characterSearch)
		if err != nil {
			return nil, err
		}
		out = characterSearch
	case XivCharacterRequest:
		resp, err := r.Do(r.XivID, r.Data...)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		log.Debugf("retry xivapi lodestone character request api reponse status code: %d", resp.StatusCode)
		if resp.StatusCode == 429 {
			durStr := resp.Header.Get("Retry-After")
			var initWait float64
			hasSuggestedRetryDur := false
			if durStr == "" {
				initWait = waitDur
			} else {
				initWait, err = strconv.ParseFloat(durStr, 64)
				if err != nil {
					log.Error(err)
					return nil, err
				}
				hasSuggestedRetryDur = true
			}
			return RetryXivApiLodestoneRequest(r, initWait, 3600, hasSuggestedRetryDur)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var character XivCharacter
		err = json.Unmarshal(respBody, &character)
		if err != nil {
			return nil, err
		}
		out = character
	default:
		return nil, fmt.Errorf("unknown lodestone request type %v", r)
	}
	return out, nil
}

func xivApiLodestoneRequestRateLimiter(rateLimit, maxRetryDuration float64, reqs chan interface{}, resps chan map[XivApiTokenMap]interface{}, tokenMaps chan XivApiTokenMap) {
	for {
		waitDur := CalcWaitDuration(rateLimit)
		log.Debug("xivApiLodestoneRequestRateLimiter is waiting for requests")
		req := <-reqs
		respToken := uuid.New().String()
		switch r := req.(type) {
		case XivCharacterSearchRequest:
			tokenMap := XivApiTokenMap{
				RequestToken:  r.Token,
				ResponseToken: respToken,
			}
			go func() {
				tokenMaps <- tokenMap
			}()
			log.Debugf("token map sent for %s", r.Name)
			resp, err := r.Do(r.Name, r.Params...)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Debugf("xivapi lodestone character search request api reponse status code: %d", resp.StatusCode)
			var outResp interface{}
			if resp.StatusCode == 429 {
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
				outResp, err = RetryXivApiLodestoneRequest(r, initWait, 3600, hasSuggestedRetryDur)
				if err != nil {
					log.Error(err)
					continue
				}
			} else {
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Error(err)
					continue
				}
				var characterSearch XivCharacterSearch
				err = json.Unmarshal(respBody, &characterSearch)
				if err != nil {
					log.Error(err)
					continue
				}
				outResp = characterSearch
			}
			log.Debugf("got response for request token %s", tokenMap.RequestToken)
			go func() {
				resps <- map[XivApiTokenMap]interface{}{tokenMap: outResp}
			}()
			<-time.After(time.Duration(waitDur) * time.Second)
		case XivCharacterRequest:
			tokenMap := XivApiTokenMap{
				RequestToken:  r.Token,
				ResponseToken: respToken,
			}
			go func() {
				tokenMaps <- tokenMap
			}()
			resp, err := r.Do(r.XivID, r.Data...)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Debugf("xivapi lodestone character search request api reponse status code: %d", resp.StatusCode)
			var outResp interface{}
			if resp.StatusCode == 429 {
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
				outResp, err = RetryXivApiLodestoneRequest(r, initWait, 3600, hasSuggestedRetryDur)
				if err != nil {
					log.Error(err)
					continue
				}
			} else {
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Error(err)
					continue
				}
				var character XivCharacter
				err = json.Unmarshal(respBody, &character)
				if err != nil {
					log.Error(err)
					continue
				}
				outResp = character
			}
			log.Debugf("got response for request token %s", tokenMap.RequestToken)
			go func() {
				resps <- map[XivApiTokenMap]interface{}{tokenMap: outResp}
			}()
			<-time.After(time.Duration(waitDur) * time.Second)
		}
	}
}
