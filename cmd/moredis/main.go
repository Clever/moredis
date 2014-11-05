package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Clever/moredis/logger"
	"github.com/Clever/moredis/moredis"
)

var (
	redisURL       string
	mongoURL       string
	mongoDBName    string
	cache          string
	params         moredis.Params
	configFilePath string
)

func init() {
	const (
		defaultFilePath = "./config.yml"
	)
	// Usage strings in PrintUsage
	flag.StringVar(&redisURL, "redis_url", "", "")
	flag.StringVar(&redisURL, "r", "", "")
	flag.StringVar(&mongoURL, "mongo_url", "", "")
	flag.StringVar(&mongoURL, "m", "", "")
	flag.StringVar(&mongoDBName, "mongo_db", "", "")
	flag.StringVar(&mongoDBName, "d", "", "")
	flag.StringVar(&cache, "cache", "", "")
	flag.StringVar(&cache, "c", "", "")
	flag.Var(&params, "params", "")
	flag.Var(&params, "p", "")
	flag.StringVar(&configFilePath, "conf_file", defaultFilePath, "")
	flag.StringVar(&configFilePath, "f", defaultFilePath, "")
}

func main() {
	flag.Usage = PrintUsage
	flag.Parse()

	// cache is the only required parameter
	if cache == "" {
		fmt.Fprintln(os.Stderr, "Missing 'cache' argument")
		flag.Usage()
		os.Exit(1)
	}

	conf, err := moredis.LoadConfig(configFilePath)
	if err != nil {
		logger.Error("Error loading config.", err)
	}

	cacheConfig, err := conf.GetCache(cache)
	if err != nil {
		logger.Error("Cache not found in config.", err)
	}

	moredis.BuildCache(cacheConfig, params, redisURL, mongoURL, mongoDBName)
}

// PrintUsage is used to replace flag.Usage, which is pretty terrible.
func PrintUsage() {
	var usage = `Usage of ./moredis:
  -c, -cache        Which cache to populate (REQUIRED)
  -d, -mongo_db     MongoDB Database, can also be set via the MONGO_DB environment variable
  -m, -mongo_url    MongoDB URL, can also be set via the MONGO_URL environment variable
  -p, -params       JSON object with params used for substitution into queries and collection names in config.yml
  -r, -redis_url    Redis URL, can also be set via the REDIS_URL environment variable
  -f, -conf_file    Config file, defaults to ./config.yml
  -h, -help         Print this usage message
`
	fmt.Fprint(os.Stderr, usage)
}
