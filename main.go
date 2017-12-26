package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	influx "github.com/MagalixTechnologies/influxdb/client/v2"
	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/ko"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/karma-go"
)

var (
	version = "[manual build]"
	usage   = "satori " + version + `

satori.

Usage:
  satori [options]
  satori -h | --help
  satori --version

Options:
  -h --help           Show this screen.
  --version           Show version.
  -c --config <path>  Use specified configuration.
                       [default: $HOME/.config/satori/satori.conf]
  --debug             Print debug messages.
  --trace             Print trace messages.
`
)

var (
	logger  *lorg.Log
	tracing bool
)

func main() {
	args, err := docopt.Parse(os.ExpandEnv(usage), nil, true, version, false)
	if err != nil {
		panic(err)
	}

	logger = lorg.NewLog()
	logger.SetIndentLines(true)
	logger.SetFormat(
		lorg.NewFormat("${time} ${level:[%s]:right:short} ${prefix}%s"),
	)

	if args["--debug"].(bool) {
		logger.SetLevel(lorg.LevelDebug)
	}

	if args["--trace"].(bool) {
		logger.SetLevel(lorg.LevelTrace)

		tracing = true
	}

	var config Config
	err = ko.Load(args["--config"].(string), &config, yaml.Unmarshal)
	if err != nil {
		fatalf(err, "unable to load config")
	}

	config.ExpandEnv()

	db, err := initDatabase(config)
	if err != nil {
		fatalf(err, "unable to establish database connection")
	}

	ticker := time.NewTicker(config.Interval)
	for {
		now := <-ticker.C

		err = tick(db, now, config.Dirs)
		if err != nil {
			logger.Error(err.Error())
		}
	}
}

func tick(db *Database, now time.Time, dirs []string) error {
	batch, err := influx.NewBatchPoints(
		influx.BatchPointsConfig{
			Database:        db.name,
			RetentionPolicy: db.rp,
		},
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to acquire new batch points",
		)
	}

	metrics := walk(dirs)

	if len(metrics) == 0 {
		warningf(nil, "collected empty payload")
		return nil
	}

	point, err := influx.NewPoint(db.measurement, nil, metrics, now)
	if err != nil {
		return karma.Format(
			err,
			"unable to acquire new influxdb point",
		)
	}

	batch.AddPoint(point)

	tracef(
		karma.Describe("metrics", traceJSON(metrics)),
		"writing metrics",
	)

	debugf(nil, "writing influxdb point")

	err = db.client.Write(batch)
	if err != nil {
		return karma.Format(
			err,
			"unable to write point to influxdb",
		)
	}

	return nil
}

func walk(dirs []string) map[string]interface{} {
	var (
		mutex   = &sync.Mutex{}
		metrics = map[string]interface{}{}
	)

	threads := &sync.WaitGroup{}
	for _, dir := range dirs {
		threads.Add(1)
		go func(dir string) {
			defer threads.Done()

			err := filepath.Walk(
				dir,
				getWalker(threads, mutex, metrics),
			)
			if err != nil {
				errorf(err,
					"unable to walk: %s", dir,
				)
			}
		}(dir)

	}

	threads.Wait()

	return metrics
}

func getWalker(
	threads *sync.WaitGroup,
	mutex *sync.Mutex,
	metrics map[string]interface{},
) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsDir() || info.Mode()&0111 == 0 {
			return nil
		}

		threads.Add(1)
		go func() {
			defer threads.Done()

			execute(path, mutex, metrics)
		}()

		return nil
	}
}

func execute(path string, mutex *sync.Mutex, metrics map[string]interface{}) {
	debugf(nil, "executing: %s", path)

	stdout, _, err := executil.Run(exec.Command(path))
	if err != nil {
		errorf(err, "unable to execute: %s", path)
		return
	}

	tracef(
		karma.Describe("stdout", string(stdout)),
		"unmarshalling output of %s",
		path,
	)

	mutex.Lock()
	defer mutex.Unlock()

	err = yaml.Unmarshal(stdout, metrics)
	if err != nil {
		errorf(err, "unable to unmarslah output of %s", path)
		return
	}
}
