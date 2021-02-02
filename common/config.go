package common

import (
	"path/filepath"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/spf13/viper"
)

type MirrorStatus struct {
	Online		bool    `json:"online"`
	OKCount		int     `json:"ok_count"`
	ErrorCount	int     `json:"error_count"`
	Hysteresis	int     `json:"hysteresis"`
}

type Mirror struct {
	Name		string  `mapstructure:"name" json:"name"`
	IsDefault	bool    `mapstructure:"default" json:"default"`
	URL		string  `mapstructure:"url" json:"url"`
	ContinentCode	string  `mapstructure:"continent_code" json:"continent_code"`
	CountryCode	string  `mapstructure:"country_code" json:"country_code"`
	Latitude	float64 `mapstructure:"latitude" json:"latitude"`
	Longitude	float64 `mapstructure:"longitude" json:"longitude"`
	Status		MirrorStatus `json:"status"`
}

type MonitorConfig struct {
	Interval	time.Duration `mapstructure:"interval"`
	Timeout		time.Duration `mapstructure:"timeout"`
	Hysteresis	int           `mapstructure:"hysteresis"`
	TLSVerify	bool          `mapstructure:"tls_verify"`
}

type Config struct {
	Debug		bool   `mapstructure:"debug"`
	Listen		string `mapstructure:"listen"`
	LogFile		string `mapstructure:"log_file"`
	MirrorListFile	string `mapstructure:"mirror_list"`
	Mirrors		map[string]*Mirror
	MMDBFile	string `mapstructure:"mmdb_file"`
	MMDB		*maxminddb.Reader
	Monitor		MonitorConfig
}

var AppConfig *Config = &Config{}


// Read main configurations from file.
//
func ReadConfig(cfgfile string) *Config {
	v := viper.New()
	v.SetConfigFile(cfgfile)
	v.SetDefault("debug", false)
	v.SetDefault("listen", "127.0.0.1:3130")
	v.SetDefault("monitor.tls_verify", true)

	err := v.ReadInConfig()
	if err != nil {
		Fatalf("Failed to read config: %v\n", err)
	}
	InfoPrintf("Read in config file.\n")

	err = v.Unmarshal(AppConfig)
	if err != nil {
		Fatalf("Failed to unmarshal config: %v\n", err)
	}

	if !AppConfig.Monitor.TLSVerify {
		WarnPrintf("TLS verification disabled!")
	}

	mlfile := AppConfig.MirrorListFile
	if mlfile == "" {
		Fatalf("Config [mirror_list] not set")
	}
	if !filepath.IsAbs(mlfile) {
		mlfile = filepath.Join(filepath.Dir(cfgfile), mlfile)
	}
	readMirrors(mlfile)

	mmdbfile := AppConfig.MMDBFile
	if mmdbfile == "" {
		Fatalf("Config [mmdb_file] not set")
	}
	if !filepath.IsAbs(mmdbfile) {
		mmdbfile = filepath.Join(filepath.Dir(cfgfile), mmdbfile)
	}
	AppConfig.MMDB, err = maxminddb.Open(mmdbfile)
	if err != nil {
		Fatalf("Failed to open MMDB: %v\n", err)
	}

	DebugPrintf("App config: %v\n", AppConfig)
	return AppConfig
}

// Read config file of mirrors.
//
func readMirrors(fname string) {
	v := viper.New()
	v.SetConfigFile(fname)
	err := v.ReadInConfig()
	if err != nil {
		Fatalf("Failed to read mirrors: %v\n", err)
	}
	InfoPrintf("Read in mirrors list.\n")

	err = v.Unmarshal(&AppConfig.Mirrors)
	if err != nil {
		Fatalf("Failed to unmarshal mirrors: %v\n", err)
	}

	var defaults []string
	for name, mirror := range AppConfig.Mirrors {
		if mirror.IsDefault {
			defaults = append(defaults, name)
		}
	}
	if len(defaults) == 0 {
		Fatalf("No default mirror set.\n")
	}
	if len(defaults) > 1 {
		Fatalf("More than one default mirrors: %v", defaults)
	}

	for name, mirror := range AppConfig.Mirrors {
		mirror.Status.Online = true
		DebugPrintf("Mirror [%s]: %v\n", name, mirror)
	}
}
