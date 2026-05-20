package config

import "flag"

type Config struct {
	Port   string
	DBPath string
	Static string
	Data   string
}

func Load() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Port, "port", "8080", "server port")
	flag.StringVar(&cfg.DBPath, "db", "./data/dailyenglish.db", "sqlite database path")
	flag.StringVar(&cfg.Static, "static", "./static", "static files directory")
	flag.StringVar(&cfg.Data, "data", "./data", "data files directory")
	flag.Parse()
	return cfg
}
