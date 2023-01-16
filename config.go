package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	BotName                             string
	MountSpreadsheetFileName            string
	MountSpreadsheetPermissionsFilepath string
	MountSpreadsheetColumnDataFilepath  string
	GoogleApiConfigRelativeFilepath     string
	AppID                               uint64
	BotUserID                           uint64
	DiscordToken                        string
	DBUsername                          string
	DBUserPassword                      string
	DBIP                                string
	DBPort                              string
	DBName                              string
	LogLevel                            uint32
}

func NewConfig(configFilepath string) (*Config, error) {
	configFile, err := os.Open(configFilepath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	rawConfig := struct {
		BotName                             string
		MountSpreadsheetFileName            string
		MountSpreadsheetPermissionsFilepath string
		MountSpreadsheetColumnDataFilepath  string
		GoogleApiConfigRelativeFilepath     string
		AppID                               uint64
		BotUserID                           uint64
		DiscordToken                        string
		DBUsername                          string
		DBUserPassword                      string
		DBIP                                string
		DBPort                              string
		DBName                              string
		LogLevel                            string
	}{}
	err = json.NewDecoder(configFile).Decode(&rawConfig)
	if err != nil {
		return nil, err
	}

	var lvl uint32
	switch rawConfig.LogLevel {
	case "Panic":
		lvl = 6
	case "Fatal":
		lvl = 5
	case "Error":
		lvl = 4
	case "Warn":
		lvl = 3
	case "Info":
		lvl = 2
	case "Debug":
		lvl = 1
	case "Trace":
		lvl = 0
	default:
		lvl = 2
	}
	return &Config{
		BotName:                             rawConfig.BotName,
		MountSpreadsheetFileName:            rawConfig.MountSpreadsheetFileName,
		MountSpreadsheetPermissionsFilepath: rawConfig.MountSpreadsheetPermissionsFilepath,
		MountSpreadsheetColumnDataFilepath:  rawConfig.MountSpreadsheetColumnDataFilepath,
		GoogleApiConfigRelativeFilepath:     rawConfig.GoogleApiConfigRelativeFilepath,
		AppID:                               rawConfig.AppID,
		BotUserID:                           rawConfig.BotUserID,
		DiscordToken:                        rawConfig.DiscordToken,
		DBUsername:                          rawConfig.DBUsername,
		DBUserPassword:                      rawConfig.DBUserPassword,
		DBIP:                                rawConfig.DBIP,
		DBPort:                              rawConfig.DBPort,
		DBName:                              rawConfig.DBName,
		LogLevel:                            lvl,
	}, nil
}
