package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsAndOverrides(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want Config
	}{
		{"database", map[string]string{"CONFIGNOSTIC_DATABASE_URL": "postgres://localhost/confignostic_dev?sslmode=disable"}, Config{"127.0.0.1:8080", 10 * time.Second, "postgres://localhost/confignostic_dev?sslmode=disable"}},
		{"defaults", nil, Config{"127.0.0.1:8080", 10 * time.Second, ""}},
		{"IPv6", map[string]string{"CONFIGNOSTIC_LISTEN_ADDR": "[::1]:8081", "CONFIGNOSTIC_SHUTDOWN_TIMEOUT": "250ms"}, Config{"[::1]:8081", 250 * time.Millisecond, ""}},
		{"ephemeral port", map[string]string{"CONFIGNOSTIC_LISTEN_ADDR": "localhost:0"}, Config{"localhost:0", 10 * time.Second, ""}},
		{"explicit public bind", map[string]string{"CONFIGNOSTIC_LISTEN_ADDR": ":8080"}, Config{":8080", 10 * time.Second, ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(func(key string) (string, bool) { value, ok := tc.env[key]; return value, ok })
			if err != nil || got != tc.want {
				t.Fatalf("Load() = %+v, %v; want %+v", got, err, tc.want)
			}
		})
	}
}

func TestLoadRejectsInvalidSettingsWithoutEchoingValues(t *testing.T) {
	for key, values := range map[string][]string{
		"CONFIGNOSTIC_LISTEN_ADDR":      {"", "localhost", "http://user:secret@localhost:8080", "localhost:65536", "localhost:-1", "localhost:+1", "localhost:http", "localhost:", "local\nhost:8080"},
		"CONFIGNOSTIC_DATABASE_URL":     {"", "secret-canary", "https://user:secret@host/db", "postgres://host/", "postgres://host/db#secret"},
		"CONFIGNOSTIC_SHUTDOWN_TIMEOUT": {"", "0s", "-1s", "10", "secret-canary"},
	} {
		for _, value := range values {
			t.Run(key+"/"+value, func(t *testing.T) {
				_, err := Load(func(name string) (string, bool) { return value, name == key })
				if err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("expected error naming %s, got %v", key, err)
				}
				if strings.Contains(err.Error(), "secret") {
					t.Fatalf("configuration error exposed its input: %v", err)
				}
			})
		}
	}
}
