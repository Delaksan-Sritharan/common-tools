// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package config loads the waste-management backend's process configuration
// from the environment. It owns all env var reads; other packages take a
// plain Config/sheets.Config value instead of reading the environment
// themselves.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/delaksan/waste-management/backend/internal/sheets"
)

// Config is the backend process configuration.
type Config struct {
	Port              string
	CORSAllowedOrigin string
	CacheTTL          time.Duration
	Sheets            sheets.Config
}

// Load reads required GOOGLE_* variables and returns an error listing every
// missing one. PORT, FRONTEND_ORIGIN, GOOGLE_SHEET_RANGE and
// CACHE_TTL_SECONDS have local defaults.
func Load() (Config, error) {
	var missing []string
	require := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := Config{
		Port:              envOrDefault("PORT", "8080"),
		CORSAllowedOrigin: envOrDefault("FRONTEND_ORIGIN", "*"),
		Sheets: sheets.Config{
			SpreadsheetID: require("GOOGLE_SHEET_ID"),
			Range:         envOrDefault("GOOGLE_SHEET_RANGE", "Sheet1!A2:H"),
			ClientID:      require("GOOGLE_OAUTH_CLIENT_ID"),
			ClientSecret:  require("GOOGLE_OAUTH_CLIENT_SECRET"),
			RefreshToken:  require("GOOGLE_OAUTH_REFRESH_TOKEN"),
		},
	}

	cacheTTL, err := durationFromSeconds("CACHE_TTL_SECONDS", 300)
	if err != nil {
		return Config{}, err
	}
	cfg.CacheTTL = cacheTTL

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func durationFromSeconds(key string, defaultSeconds int) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(defaultSeconds) * time.Second, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if seconds < 0 {
		return 0, fmt.Errorf("%s must be non-negative", key)
	}
	return time.Duration(seconds) * time.Second, nil
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
