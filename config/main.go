package config

import (
	"log"
	"path/filepath"

	"github.com/oschwald/maxminddb-golang"
	"github.com/spf13/viper"
)

type Mirror struct {
	Name		string  `mapstructure:"name" json:"name"`
	IsDefault	bool    `mapstructure:"default" json:"default"`
	URL		string  `mapstructure:"url" json:"url"`
	ContinentCode	string  `mapstructure:"continent_code" json:"continent_code"`
	CountryCode	string  `mapstructure:"country_code" json:"country_code"`
	Latitude	float64 `mapstructure:"latitude" json:"latitude"`
	Longitude	float64 `mapstructure:"longitude" json:"longitude"`
	// For mirror determination & monitor
	IsOffline	bool    `json:"offline"`
}

type Config struct {
	Debug		bool   `mapstructure:"debug"`
	Listen		string `mapstructure:"listen"`
	LogFile		string `mapstructure:"log_file"`
	MirrorListFile	string `mapstructure:"mirror_list"`
	Mirrors		map[string]*Mirror
	MMDBFile	string `mapstructure:"mmdb_file"`
	MMDB		*maxminddb.Reader
}

var AppConfig *Config = &Config{}


// Read main configurations from file.
//
func ReadConfig(cfgfile string) *Config {
	v := viper.New()
	v.SetConfigFile(cfgfile)
	v.SetDefault("debug", false)
	v.SetDefault("listen", "127.0.0.1:3130")

	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v\n", err)
	}
	log.Println("Read in config file.")

	err = v.Unmarshal(AppConfig)
	if err != nil {
		log.Fatalf("Failed to unmarshal config: %v\n", err)
	}

	mlfile := AppConfig.MirrorListFile
	if mlfile == "" {
		log.Fatal("Config [mirror_list] not set")
	}
	if !filepath.IsAbs(mlfile) {
		mlfile = filepath.Join(filepath.Dir(cfgfile), mlfile)
	}
	ReadMirrors(mlfile)

	mmdbfile := AppConfig.MMDBFile
	if mmdbfile == "" {
		log.Fatal("Config [mmdb_file] not set")
	}
	if !filepath.IsAbs(mmdbfile) {
		mmdbfile = filepath.Join(filepath.Dir(cfgfile), mmdbfile)
	}
	AppConfig.MMDB, err = maxminddb.Open(mmdbfile)
	if err != nil {
		log.Fatalf("Failed to open MMDB: %v\n", err)
	}

	return AppConfig
}

// Read config file of mirrors.
//
func ReadMirrors(fname string) {
	v := viper.New()
	v.SetConfigFile(fname)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Failed to read mirrors: %v\n", err)
	}
	log.Println("Read in mirrors list.")

	err = v.Unmarshal(&AppConfig.Mirrors)
	if err != nil {
		log.Fatalf("Failed to unmarshal mirrors: %v\n", err)
	}

	var defaults []string
	for name, mirror := range AppConfig.Mirrors {
		if mirror.IsDefault {
			defaults = append(defaults, name)
		}
	}
	if len(defaults) == 0 {
		log.Fatalf("No default mirror set.\n")
	}
	if len(defaults) > 1 {
		log.Fatalf("More than one default mirrors: %v", defaults)
	}
}
