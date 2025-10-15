package main

import (
	"component-monitor/pkg/types"
	"errors"
	"flag"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Options struct {
	ConfigPath  string
	Port        string
	DatabaseDSN string
}

func NewOptions() *Options {
	opts := &Options{}

	flag.StringVar(&opts.ConfigPath, "config", "", "Path to config file")
	flag.StringVar(&opts.Port, "port", "8080", "Port to listen on")
	flag.StringVar(&opts.DatabaseDSN, "dsn", "", "PostgreSQL DSN connection string")
	flag.Parse()

	return opts
}

func (o *Options) Validate() error {
	if o.ConfigPath == "" {
		return errors.New("config path is required (use --config flag)")
	}

	if _, err := os.Stat(o.ConfigPath); os.IsNotExist(err) {
		return errors.New("config file does not exist: " + o.ConfigPath)
	}

	if o.Port == "" {
		return errors.New("port cannot be empty")
	}

	if o.DatabaseDSN == "" {
		return errors.New("database DSN is required (use --dsn flag)")
	}

	return nil
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	opts := NewOptions()

	if err := opts.Validate(); err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	log.Infof("Loading config from %s", opts.ConfigPath)

	configFile, err := os.ReadFile(opts.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	log.Infof("Loaded configuration with %d components", len(config.Components))

	// Connect to database
	log.Info("Connecting to PostgreSQL database")
	db, err := gorm.Open(postgres.Open(opts.DatabaseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	server := NewServer(&config, db)

	addr := ":" + opts.Port
	if err := server.Start(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
