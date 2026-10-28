// Package config validates process configuration before opening a listener.
package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr      string
	ShutdownTimeout time.Duration
	DatabaseURL     string
}

// Load accepts a lookup function so tests never mutate process-wide environment.
// An explicitly empty setting is an error, rather than a silent default.
func Load(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{ListenAddr: "127.0.0.1:8080", ShutdownTimeout: 10 * time.Second}
	if value, set := lookup("CONFIGNOSTIC_LISTEN_ADDR"); set {
		cfg.ListenAddr = value
	}
	host, port, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil || strings.ContainsAny(host, " \t\r\n\x00/\\") || port == "" {
		return Config{}, errors.New("CONFIGNOSTIC_LISTEN_ADDR must be host:port with a numeric port")
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return Config{}, errors.New("CONFIGNOSTIC_LISTEN_ADDR port must be between 0 and 65535")
		}
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return Config{}, errors.New("CONFIGNOSTIC_LISTEN_ADDR port must be between 0 and 65535")
	}
	if value, set := lookup("CONFIGNOSTIC_SHUTDOWN_TIMEOUT"); set {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return Config{}, errors.New("CONFIGNOSTIC_SHUTDOWN_TIMEOUT must be a positive duration, for example 10s")
		}
		cfg.ShutdownTimeout = duration
	}
	if value, set := lookup("CONFIGNOSTIC_DATABASE_URL"); set {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Path == "" || parsed.Path == "/" || parsed.Fragment != "" || strings.ContainsAny(value, "\r\n\x00") {
			return Config{}, errors.New("CONFIGNOSTIC_DATABASE_URL must be a PostgreSQL URL with a database name")
		}
		cfg.DatabaseURL = value
	}
	return cfg, nil
}
