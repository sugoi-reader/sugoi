package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
)

var config SugoiConfig

type SugoiConfig struct {
	Debug               bool
	CacheThumbnails     bool
	CacheDir            string
	DatabaseDir         string
	ServerHost          string
	ServerPort          int
	DirVars             map[string]string
	SessionCookieName   string
	SessionCookieMaxAge int
	SessionCookieKey    []byte
	Users               map[string]string
	// DefaultCoverFileName string
}

func (c SugoiConfig) CacheFile(elem ...string) string {
	params := []string{c.CacheDir}
	params = append(params, elem...)
	return path.Join(params...)
}

func (c SugoiConfig) Save(to string) error {
	var err error
	configFile, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return err
	}
	defer configFile.Close()

	b, err := json.MarshalIndent(c, "", "\t")

	if err != nil {
		return err
	}

	_, err = configFile.Write(b)

	return err
}

func (c SugoiConfig) Export() (string, error) {
	b, err := json.MarshalIndent(c, "", "\t")

	if err != nil {
		return "", err
	}

	return string(b), nil
}

func InitializeConfig() error {
	var err error
	configFile, err := os.Open(configPath)

	if err != nil {
		return err
	}

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return err
	}

	// if config.DefaultCoverFileName == "" {
	// 	config.DefaultCoverFileName = "01.png"
	// }

	if config.SessionCookieMaxAge <= 0 {
		return fmt.Errorf("SessionMaxAge should be greater than 0")
	}

	if len(config.SessionCookieKey) < 32 {
		fmt.Println("SessionKey should be a base64 encoded secret byte array with at least 32 bytes")
		fmt.Println("Like this:")
		b := make([]byte, 32)
		rand.Read(b)
		sEnc := base64.StdEncoding.EncodeToString(b)
		fmt.Println(sEnc)
		return fmt.Errorf("Update your configuration file with a valid value, then run sugoi again")
	}

	return nil
}
