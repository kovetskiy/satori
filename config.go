package main

import (
	"os"
	"time"
)

type Config struct {
	Interval time.Duration `required:"true"`

	Database struct {
		Address         string `required:"true"`
		RetentionPolicy string `yaml:"retention_policy" required:"true"`
		Measurement     string `required:"true"`
	} `required:"true"`

	Dirs []string `required:"true"`
}

func (config *Config) ExpandEnv() {
	for i, dir := range config.Dirs {
		config.Dirs[i] = os.ExpandEnv(dir)
	}
}
