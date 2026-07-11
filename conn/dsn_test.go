package conn

import (
	"testing"
	"time"
)

func TestParseDSNUsesFirstHost(t *testing.T) {
	cfg, err := ParseDSN("clickhouse://host1:9000,host2:9001/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "host1:9000" {
		t.Errorf("Addr = %q, want host1:9000", cfg.Addr)
	}
}

func TestParseDSNCompressionMapping(t *testing.T) {
	cfg, err := ParseDSN("clickhouse://host/db?compress=lz4")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Compression != CompressionEnabled {
		t.Errorf("Compression = %v, want enabled", cfg.Compression)
	}
}

func TestParseDSNEmptyCompression(t *testing.T) {
	cfg, err := ParseDSN("clickhouse://host/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Compression != CompressionDisabled {
		t.Errorf("Compression = %v, want disabled", cfg.Compression)
	}
}

func TestParseDSNSettings(t *testing.T) {
	cfg, err := ParseDSN("clickhouse://host/db?max_threads=4")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Settings) != 1 || cfg.Settings[0].Key != "max_threads" || cfg.Settings[0].Value != "4" {
		t.Errorf("Settings = %v, want [{max_threads 4}]", cfg.Settings)
	}
}

func TestParseDSNDuration(t *testing.T) {
	cfg, err := ParseDSN("clickhouse://host/db?dial_timeout=5s")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("DialTimeout = %v, want 5s", cfg.DialTimeout)
	}
}
