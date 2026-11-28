// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2021-2026 The DragonFly Project.

package common

import (
	"net/url"
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
func ReadConfig(cfgfile string) *Config {
	_, err := toml.DecodeFile(cfgfile, &AppConfig)
	if err != nil {
		Fatalf("Failed to read config file `%s`: %v\n", cfgfile, err)
	}
	InfoPrintf("Read in config file: %s\n", cfgfile)

	if AppConfig.Monitor.Workers <= 0 {
		Fatalf("Config [monitor.workers] = %d <= 0\n",
			AppConfig.Monitor.Workers)
	}

	if !AppConfig.Monitor.TLSVerify {
		WarnPrintf("TLS verification disabled! THIS IS INSECURE!!!")
	}

	mlfile := AppConfig.MirrorListFile
	if mlfile == "" {
		Fatalf("Config [mirror_list] not set")
	}
	if !filepath.IsAbs(mlfile) {
		mlfile = filepath.Join(filepath.Dir(cfgfile), mlfile)
	}
	readMirrors(mlfile)

	switch strings.ToLower(AppConfig.MMDBType) {
	case "db-ip", "dbip":
		AppConfig.MMDB.Type = MMDB_DBIP
	case "maxmind":
		AppConfig.MMDB.Type = MMDB_MAXMIND
	default:
		Fatalf("Config [mmdb_type] invalid: %v", AppConfig.MMDBType)
	}

	mmdbfile := AppConfig.MMDBFile
	if mmdbfile == "" {
		Fatalf("Config [mmdb_file] not set")
	}
	if !filepath.IsAbs(mmdbfile) {
		mmdbfile = filepath.Join(filepath.Dir(cfgfile), mmdbfile)
	}
	AppConfig.MMDB.DB, err = maxminddb.Open(mmdbfile)
	if err != nil {
		Fatalf("Failed to open MMDB: %v\n", err)
	}

	DebugPrintf("App config: %+v\n", AppConfig)
	return AppConfig
}

// Read config file of mirrors.
func readMirrors(fname string) {
	if _, err := toml.DecodeFile(fname, &AppConfig.Mirrors); err != nil {
		Fatalf("Failed to read mirrors file `%s`: %v\n", fname, err)
	}
	InfoPrintf("Read mirrors list from file: %s\n", fname)

	var defaults []string
	for name, mirror := range AppConfig.Mirrors {
		// Some mirrors may return 404 if there is no trailing slash.
		if !strings.HasSuffix(mirror.URL, "/") {
			mirror.URL += "/"
		}

		u, err := url.Parse(mirror.URL)
		if err != nil {
			Fatalf("Mirror [%s] URL invalid: %v\n",
				name, mirror.URL)
		}
		switch u.Scheme {
		case "http", "https", "ftp":
			break
		default:
			Fatalf("Mirror [%s] URL unsupported: %v\n",
				name, mirror.URL)
		}

		if mirror.ContinentCode == "" || mirror.CountryCode == "" ||
			mirror.Latitude == 0 || mirror.Longitude == 0 {
			Fatalf("Mirror [%s] location incomplete\n", name)
		}

		mirror.Status.Online = true
		DebugPrintf("Mirror [%s]: %+v\n", name, mirror)

		if mirror.IsDefault {
			defaults = append(defaults, name)
			InfoPrintf("Default mirror: %s\n", name)
		}
	}

	if len(defaults) == 0 {
		Fatalf("No default mirror set.\n")
	}
	if len(defaults) > 1 {
		Fatalf("More than one default mirrors: %v", defaults)
	}
}
