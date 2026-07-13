package pool

import (
	"testing"

	"github.com/ddukki/scorch/conn"
)

func TestParsePoolDSNMultiHost(t *testing.T) {
	cfg, err := ParsePoolDSN("clickhouse://host1:9000,host2:9001/db")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"host1:9000", "host2:9001"}
	if len(cfg.Addrs) != len(expected) {
		t.Fatalf("Addrs = %v, want %v", cfg.Addrs, expected)
	}
	for i, a := range cfg.Addrs {
		if a != expected[i] {
			t.Errorf("Addrs[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestParsePoolDSNSingleHost(t *testing.T) {
	cfg, err := ParsePoolDSN("clickhouse://host1:9000/db")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "host1:9000" {
		t.Errorf("Addrs = %v, want [host1:9000]", cfg.Addrs)
	}
}

func TestParsePoolDSNCompression(t *testing.T) {
	cfg, err := ParsePoolDSN("clickhouse://host/db?compress=lz4")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Config.Compression != conn.CompressionEnabled {
		t.Errorf("Compression = %v, want enabled", cfg.Config.Compression)
	}
}

func TestParsePoolDSNSettings(t *testing.T) {
	cfg, err := ParsePoolDSN("clickhouse://host/db?max_threads=4")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Config.Settings) != 1 || cfg.Config.Settings[0].Key != "max_threads" || cfg.Config.Settings[0].Value != "4" {
		t.Errorf("Settings = %v, want [{max_threads 4}]", cfg.Config.Settings)
	}
}

func TestParsePoolDSNError(t *testing.T) {
	_, err := ParsePoolDSN("http://host/db")
	if err == nil {
		t.Fatal("expected error for wrong scheme")
	}
}
