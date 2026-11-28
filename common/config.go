// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package common

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/oschwald/maxminddb-golang"
)

type MirrorStatus struct {
	Online     bool `json:"online"`
	OKCount    int  `json:"ok_count"`
	ErrorCount int  `json:"error_count"`
	Hysteresis int  `json:"hysteresis"`
}

type Mirror struct {
	Name          string       `toml:"name" json:"name"`
	IsDefault     bool         `toml:"default" json:"default"`
	URL           string       `toml:"url" json:"url"`
	ContinentCode string       `toml:"continent_code" json:"continent_code"`
	CountryCode   string       `toml:"country_code" json:"country_code"`
	Latitude      float64      `toml:"latitude" json:"latitude"`
	Longitude     float64      `toml:"longitude" json:"longitude"`
	Status        MirrorStatus `json:"status"`
}

type MMDBConfig struct {
	Type int
	DB   *maxminddb.Reader
}

type MonitorConfig struct {
	Workers     int    `toml:"workers"`
	Interval    int    `toml:"interval"` // seconds
	Timeout     int    `toml:"timeout"`  // seconds
	Hysteresis  int    `toml:"hysteresis"`
	TLSVerify   bool   `toml:"tls_verify"`
	UserAgent   string `toml:"user_agent"`
	NotifyExec  string `toml:"notify_exec"`
	ExecTimeout int    `toml:"exec_timeout"` // seconds
}

type Config struct {
	Debug          bool     `toml:"debug"`
	Listen         string   `toml:"listen"`
	TrustedProxies []string `toml:"trusted_proxies"`
	MirrorListFile string   `toml:"mirror_list"`
	Mirrors        map[string]*Mirror
	MMDBType       string `toml:"mmdb_type"`
	MMDBFile       string `toml:"mmdb_file"`
	MMDB           MMDBConfig
	Monitor        MonitorConfig
}

const (
	MMDB_DBIP = iota
	MMDB_MAXMIND
)

var AppConfig = &Config{
	Listen: "127.0.0.1:3130",
	Monitor: MonitorConfig{
		Workers:     10,
		Interval:    3600, // hourly
		Timeout:     5,
		Hysteresis:  3,
		TLSVerify:   true,
		UserAgent:   AppName + "/" + Version,
		ExecTimeout: 3,
	},
}

// Read main configurations from file.
func ReadConfig(cfgfile string, logger *slog.Logger) *Config {
	logger = logger.With("component", "common")

	_, err := toml.DecodeFile(cfgfile, &AppConfig)
	if err != nil {
		logger.Error("failed to read config", "file", cfgfile, "error", err)
		os.Exit(1)
	}
	logger.Info("read config", "file", cfgfile)

	if AppConfig.Monitor.Workers <= 0 {
		logger.Error("invalid [monitor.workers]", "value", AppConfig.Monitor.Workers)
		os.Exit(1)
	}

	if !AppConfig.Monitor.TLSVerify {
		logger.Warn("TLS verification disabled! THIS IS INSECURE!!!")
	}

	mlfile := AppConfig.MirrorListFile
	if mlfile == "" {
		logger.Error("[mirror_list] required but not set")
		os.Exit(1)
	}
	if !filepath.IsAbs(mlfile) {
		mlfile = filepath.Join(filepath.Dir(cfgfile), mlfile)
	}
	readMirrors(mlfile, logger)

	switch strings.ToLower(AppConfig.MMDBType) {
	case "db-ip", "dbip":
		AppConfig.MMDB.Type = MMDB_DBIP
	case "maxmind":
		AppConfig.MMDB.Type = MMDB_MAXMIND
	default:
		logger.Error("invalid [mmdb_type]", "value", AppConfig.MMDBType)
		os.Exit(1)
	}

	mmdbfile := AppConfig.MMDBFile
	if mmdbfile == "" {
		logger.Error("[mmdb_file] required but not set")
		os.Exit(1)
	}
	if !filepath.IsAbs(mmdbfile) {
		mmdbfile = filepath.Join(filepath.Dir(cfgfile), mmdbfile)
	}
	AppConfig.MMDB.DB, err = maxminddb.Open(mmdbfile)
	if err != nil {
		logger.Error("failed to open MMDB", "file", mmdbfile, "error", err)
		os.Exit(1)
	}

	logger.Debug("loaded config", "value", AppConfig)
	return AppConfig
}

// Read config file of mirrors.
func readMirrors(fname string, logger *slog.Logger) {
	if _, err := toml.DecodeFile(fname, &AppConfig.Mirrors); err != nil {
		logger.Error("failed to read mirrors", "file", fname, "error", err)
		os.Exit(1)
	}
	logger.Info("Read mirrors", "file", fname)

	var defaults []string
	for name, mirror := range AppConfig.Mirrors {
		// Some mirrors may return 404 if there is no trailing slash.
		if !strings.HasSuffix(mirror.URL, "/") {
			mirror.URL += "/"
		}

		u, err := url.Parse(mirror.URL)
		if err != nil {
			logger.Error("invalid mirror", "name", name, "url", mirror.URL)
			os.Exit(1)
		}
		switch u.Scheme {
		case "http", "https", "ftp":
			break
		default:
			logger.Error("unsupported mirror", "name", name, "url", mirror.URL)
			os.Exit(1)
		}

		if mirror.ContinentCode == "" || mirror.CountryCode == "" ||
			mirror.Latitude == 0 || mirror.Longitude == 0 {
			logger.Error("mirror location incomplete", "name", name)
			os.Exit(1)
		}

		mirror.Status.Online = true
		logger.Debug("added mirror", "name", name, "info", mirror)

		if mirror.IsDefault {
			defaults = append(defaults, name)
			logger.Info("default mirror", "name", name)
		}
	}

	if len(defaults) == 0 {
		logger.Error("default mirror required but not set")
		os.Exit(1)
	}
	if len(defaults) > 1 {
		logger.Error("found multiple default mirrors", "mirrors", defaults)
		os.Exit(1)
	}
}
