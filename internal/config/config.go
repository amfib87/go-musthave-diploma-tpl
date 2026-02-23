package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress    string
	DBURI         string
	AccSystemAddr string
}

func Initialize() *Config {
	return &Config{
		RunAddress:    "",
		DBURI:         "",
		AccSystemAddr: "",
	}
}

func (cf *Config) ParseFlag() {
	flag.StringVar(&cf.RunAddress, "a", "", "address for start server")
	flag.StringVar(&cf.DBURI, "d", "", "address for connect to DB")
	flag.StringVar(&cf.AccSystemAddr, "r", "", "address of the accrual calculation system")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	if envRunAddress, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cf.RunAddress = envRunAddress
	}

	if envDBURI, ok := os.LookupEnv("DATABASE_URI"); ok {
		cf.DBURI = envDBURI
	}

	if envAccSystemAddr, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cf.AccSystemAddr = envAccSystemAddr
	}
}
