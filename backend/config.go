package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

type ConfigYaml struct {
	LogLevel string `yaml:"log_level"`
	Database struct {
		Host     string `yaml:"host"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"database"`
	Cache struct {
		Expiration string `yaml:"expiration"`
		Purge      string `yaml:"purge"`
	} `yaml:"cache"`
	ClientSession struct {
		JwtSignature string `yaml:"jwt_signature"`
		Expire       string `yaml:"expire"`
	} `yaml:"client_session"`
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
}

type CacheConfig struct {
	Expiration time.Duration
	Purge      time.Duration
}

type ConfigStruct struct {
	ConfigYaml
	LogLevel      zerolog.Level
	SessionExpire time.Duration
	Cache         CacheConfig
}

var config ConfigStruct

var logger zerolog.Logger

type specificLevelWriter struct {
	io.Writer
	Level zerolog.Level
}

func (w specificLevelWriter) WriteLevel(l zerolog.Level, p []byte) (int, error) {
	if l >= w.Level {
		return w.Write(p)
	} else {
		return len(p), nil
	}
}

type Payload struct {
	jwt.RegisteredClaims
	CustomClaims map[string]any
}

func (config ConfigStruct) signJWT(val any) (string, error) {
	valMap, err := strucToMap(val)

	if err != nil {
		return "", err
	}

	payload := Payload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.SessionExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		CustomClaims: valMap,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return t.SignedString([]byte(config.ClientSession.JwtSignature))
}

func loadConfig() ConfigStruct {
	config := ConfigYaml{}

	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.Panic().Msgf("Error opening config-file: %q", err)
	}

	reader := bytes.NewReader(yamlFile)

	dec := yaml.NewDecoder(reader)
	dec.KnownFields(true)
	err = dec.Decode(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config-file: %q", err.Error())
		os.Exit(1)
	}

	if logLevel, err := zerolog.ParseLevel(config.LogLevel); err != nil {
		panic(fmt.Errorf("can't parse log-level: %v", err))
	} else {
		var configStruct ConfigStruct

		if session_expire, err := time.ParseDuration(config.ClientSession.Expire); err != nil {
			fmt.Fprintf(os.Stderr, `Error Parsing "client_session.expire": %q`, err.Error())
			os.Exit(1)
		} else if cacheExpire, err := time.ParseDuration(config.Cache.Expiration); err != nil {
			fmt.Fprintf(os.Stderr, `Error Parsing "cache.expiration": %q`, err.Error())
			os.Exit(1)
		} else if cachePurge, err := time.ParseDuration(config.Cache.Purge); err != nil {
			fmt.Fprintf(os.Stderr, `Error Parsing "cache.purge": %q`, err.Error())
			os.Exit(1)
		} else {
			configStruct = ConfigStruct{
				ConfigYaml:    config,
				LogLevel:      logLevel,
				SessionExpire: session_expire,
				Cache: CacheConfig{
					Expiration: cacheExpire,
					Purge:      cachePurge,
				},
			}
		}

		return configStruct
	}
}

func init() {
	config = loadConfig()

	// try to set the log-level
	zerolog.SetGlobalLevel(config.LogLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// create the console output
	outputConsole := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		},
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s", i)
		},
		NoColor: true,
	}

	// create the logfile output
	outputLog := &lumberjack.Logger{
		Filename:  "logs/livestreamScheduler.log",
		MaxAge:    7,
		LocalTime: true,
	}

	// create a multi-output-writer
	multi := zerolog.MultiLevelWriter(
		specificLevelWriter{
			Writer: outputConsole,
			Level:  config.LogLevel,
		},
		specificLevelWriter{
			Writer: outputLog,
			Level:  config.LogLevel,
		},
	)

	// create a logger-instance
	logger = zerolog.New(multi).With().Timestamp().Logger()
}
