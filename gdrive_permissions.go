package main

import (
	"encoding/json"
	"io/ioutil"
	"regexp"

	"google.golang.org/api/drive/v3"
)

func isGmailEmailAddress(email string) bool {
	rexp := regexp.MustCompile(`^.+@gmail.com$`)
	return rexp.Match([]byte(email))
}

func GetPermissions(mountSpreadsheetPermissionsFilepath string) ([]*drive.Permission, error) {
	rawJson, err := ioutil.ReadFile(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		return nil, err
	}
	var perms []*drive.Permission
	err = json.Unmarshal(rawJson, &perms)
	if err != nil {
		return nil, err
	}
	return perms, nil
}
