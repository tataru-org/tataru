package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/disgoorg/log"
)

const (
	XivApiRootUrl            = "https://xivapi.com"
	XivApiCharacterUrl       = XivApiRootUrl + "/character"
	XivApiCharacterSearchUrl = XivApiCharacterUrl + "/search"
)

type XivCharacterData string

const (
	XivCharacterDataAchievements       XivCharacterData = "AC"
	XivCharacterDataFriendsList        XivCharacterData = "FR"
	XivCharacterDataFreeCompany        XivCharacterData = "FC"
	XivCharacterDataFreeCompanyMembers XivCharacterData = "FCM"
	XivCharacterDataMountsMinions      XivCharacterData = "MIMO"
	XivCharacterDataPvpTeam            XivCharacterData = "PVP"
)

type XivApiQueryParam struct {
	Name  string
	Value string
}
type XivApiClient struct {
	c   *http.Client
	key string
}

func NewXivApiClient(apiKey string, client *http.Client) *XivApiClient {
	c := client
	if c == nil {
		c = &http.Client{Timeout: time.Duration(60) * time.Second}
	} else if c == http.DefaultClient {
		c.Timeout = time.Duration(60) * time.Second
	}
	return &XivApiClient{
		c:   c,
		key: apiKey,
	}
}

type XivCharacterSearchRequest struct {
	Token  string
	Name   string
	Params []XivApiQueryParam
	Do     func(string, ...XivApiQueryParam) (*http.Response, error)
}

type XivCharacterRequest struct {
	Token string
	XivID string
	Data  []XivCharacterData
	Do    func(string, ...XivCharacterData) (*http.Response, error)
}

/*
	required params:
		name: Must replace spaces with '+'
	optional params:
		server
		page
*/
func (xiv *XivApiClient) SearchForCharacter(name string, params ...XivApiQueryParam) (*http.Response, error) {
	queryName := strings.ReplaceAll(name, " ", "+")
	url := fmt.Sprintf("%s?private_key=%s", XivApiCharacterSearchUrl, xiv.key)
	url = fmt.Sprintf("%s&name=%s", url, queryName)
	for i := 0; i < len(params); i++ {
		url = fmt.Sprintf("%s&%s=%s", url, params[i].Name, params[i].Value)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	log.Debug("sending character search request for %s", name)
	return xiv.c.Do(req)

}

/*
	required params:
		xivid
	optional params:
		data:
			Implemented data structures:
				XivCharacterDataFreeCompanyMembers
				XivCharacterDataMountsMinions

*/
func (xiv *XivApiClient) GetCharacter(xivid string, data ...XivCharacterData) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s?private_key=%s", XivApiCharacterUrl, xivid, xiv.key)
	for i := 0; i < len(data); i++ {
		if i == 0 {
			url = fmt.Sprintf("%s&data=%s", url, data[i])
		} else {
			url = fmt.Sprintf("%s,%s", url, data[i])
		}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	log.Debug("sending character request for %s", xivid)
	return xiv.c.Do(req)

}
