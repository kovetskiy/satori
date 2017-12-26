package main

import (
	"net/url"
	"strings"
	"time"

	influx "github.com/MagalixTechnologies/influxdb/client/v2"
	"github.com/reconquest/karma-go"
)

type Database struct {
	client      influx.Client
	name        string
	measurement string
	rp          string
}

func initDatabase(config Config) (*Database, error) {
	uri, err := url.Parse(config.Database.Address)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to parse dsn",
		)
	}

	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: config.Database.Address,
	})
	if err != nil {
		return nil, err
	}

	infof(
		karma.Describe("dsn", config.Database.Address),
		"connecting to influx database",
	)

	for {
		pong, version, err := client.Ping(time.Second)
		if err != nil {
			errorf(err, "unable to ping influx database, reconnecting...")
		} else {
			infof(
				karma.
					Describe("ping", pong).
					Describe("version", version),
				"connected to influx database",
			)
			break
		}

		time.Sleep(time.Second)
	}

	var db Database
	db.client = client
	db.name = strings.TrimPrefix(uri.Path, "/")
	db.rp = config.Database.RetentionPolicy
	db.measurement = config.Database.Measurement

	return &db, nil
}
