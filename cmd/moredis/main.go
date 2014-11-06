package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Clever/moredis/logger"
	"github.com/Clever/moredis/moredis"
)

// Default database connection parameters
const (
	DefaultMongoURL = "localhost:27017"
	DefaultRedisURL = "localhost:6379"
)

var (
	redisURL       string
	mongoURL       string
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

	// grab connection from env or default if not in flags
	mongoURL = FlagEnvOrDefault(mongoURL, "MONGO_URL", DefaultMongoURL)
	redisURL = FlagEnvOrDefault(redisURL, "REDIS_URL", DefaultRedisURL)

	conf, err := moredis.LoadConfig(configFilePath)
	if err != nil {
		logger.Error("Error loading config.", err)
		os.Exit(1)
	}

	cacheConfig, err := conf.GetCache(cache)
	if err != nil {
		logger.Error("Cache not found in config.", err)
		os.Exit(1)
	}

	if err := moredis.BuildCache(cacheConfig, params, redisURL, mongoURL); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

// PrintUsage is used to replace flag.Usage, which is pretty terrible.
func PrintUsage() {
	var usage = `Usage of ./moredis:
  -c, -cache        Which cache to populate (REQUIRED)
  -m, -mongo_url    MongoDB URL, can also be set via the MONGO_URL environment variable
  -p, -params       JSON object with params used for substitution into queries and collection names in config.yml
  -r, -redis_url    Redis URL, can also be set via the REDIS_URL environment variable
  -f, -conf_file    Config file, defaults to ./config.yml
  -h, -help         Print this usage message
`
	fmt.Fprint(os.Stderr, usage)
}

// FlagEnvOrDefault takes in the value returned from flag parsing, and environment variable to check, and a default value.
// If the flag was not set, it tries to retrieve the value from environment variables.  If it is not found there it returns
// the default value.
func FlagEnvOrDefault(flagVal, envVar, defaultValue string) string {
	if flagVal != "" {
		return flagVal
	}
	if fromEnv := os.Getenv(envVar); fromEnv != "" {
		return fromEnv
	}
	return defaultValue
}
