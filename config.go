package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	BotName                        string
	MountSpreadsheetFileName       string
	MountSpreadsheetTitle          string
	GoogleDriveDestinationFolderId string
	DiscordToken                   string
	XivapiApiKey                   string
	DBUsername                     string
	DBUserPassword                 string
	DBIP                           string
	DBPort                         string
	DBName                         string
	LogLevel                       uint32
}

func NewConfig(configFilepath string) (*Config, error) {
	configFile, err := os.Open(configFilepath)
	if err != nil {
		return nil, fmt.Errorf("os.Open() error: [%w]", err)
	}
	defer configFile.Close()

	rawConfig := struct {
		BotName                        string
		MountSpreadsheetFileName       string
		MountSpreadsheetTitle          string
		GoogleDriveDestinationFolderId string
		DiscordToken                   string
		XivapiApiKey                   string
		DBUsername                     string
		DBUserPassword                 string
		DBIP                           string
		DBPort                         string
		DBName                         string
		LogLevel                       string
	}{}
	err = json.NewDecoder(configFile).Decode(&rawConfig)
	if err != nil {
		return nil, fmt.Errorf("json.NewDecoder().Decode() error: [%w]", err)
	}

	var lvl uint32
	switch rawConfig.LogLevel {
	case "panic":
		lvl = 6
	case "fatal":
		lvl = 5
	case "error":
		lvl = 4
	case "warn":
		lvl = 3
	case "info":
		lvl = 2
	case "debug":
		lvl = 1
	case "trace":
		lvl = 0
	default:
		lvl = 2
	}
	return &Config{
		BotName:                        rawConfig.BotName,
		MountSpreadsheetFileName:       rawConfig.MountSpreadsheetFileName,
		MountSpreadsheetTitle:          rawConfig.MountSpreadsheetTitle,
		GoogleDriveDestinationFolderId: rawConfig.GoogleDriveDestinationFolderId,
		DiscordToken:                   rawConfig.DiscordToken,
		XivapiApiKey:                   rawConfig.XivapiApiKey,
		DBUsername:                     rawConfig.DBUsername,
		DBUserPassword:                 rawConfig.DBUserPassword,
		DBIP:                           rawConfig.DBIP,
		DBPort:                         rawConfig.DBPort,
		DBName:                         rawConfig.DBName,
		LogLevel:                       lvl,
	}, nil
}
