// Copyright (c) 2024 Tim <tbckr>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tbckr/trident/pkg/pap"
)

const (
	appName               = "trident"
	defaultDirPermissions = 0755

	// App specific keys
	ConfigKeyPapLevel              = "papLevel"
	ConfigKeyDisableDomainBrackets = "disableDomainBrackets"

	// Plugin specific keys
	ConfigKeySecurityTrailsApiKey = "securitytrails.apiKey"
)

var (
	ErrApiKeyNotSet = errors.New("API key not set")
)

type Config struct {
	*viper.Viper
}

func New(userconfigdir func() (string, error)) (*Config, error) {
	viperConfig := viper.New()

	configDir, err := userconfigdir()
	if err != nil {
		return nil, err
	}
	var appConfigDir string
	appConfigDir, err = getAppConfigDirPath(configDir, appName)
	if err != nil {
		return nil, err
	}

	viper.AddConfigPath(appConfigDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err = viperConfig.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// Config file not found
			// Skip error if the file does not exist
			return &Config{
				Viper: viperConfig,
			}, nil
		}
		// Config file was found but another error was produced
		return nil, err
	}
	return &Config{
		Viper: viperConfig,
	}, nil
}

func getAppConfigDirPath(elem ...string) (string, error) {
	appPath := filepath.Join(elem...)
	// if dir does not exist, create it
	if _, err := os.Stat(appPath); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(appPath, defaultDirPermissions); err != nil {
			return "", err
		}
	}
	return appPath, nil
}

func (c *Config) GetEnvironmentPapLevel() (pap.PapLevel, error) {
	// No check, because the default value is WHITE
	val := c.GetString(ConfigKeyPapLevel)
	environmentPapLevel, err := pap.GetLevel(val)
	if err != nil {
		return pap.LevelWhite, err
	}
	return environmentPapLevel, nil
}

func (c *Config) GetDisableDomainBrackets() bool {
	// No check, because the default value is false
	return c.GetBool(ConfigKeyDisableDomainBrackets)
}

func (c *Config) GetSecurityTrailsApiKey() (string, error) {
	if !c.IsSet(ConfigKeySecurityTrailsApiKey) {
		return "", &ApiKeyNotSetError{
			Plugin: "SecurityTrails",
		}
	}
	return c.GetString(ConfigKeySecurityTrailsApiKey), nil
}
