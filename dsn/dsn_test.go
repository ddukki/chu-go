package dsn

import (
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

func TestMinimalDSN(t *testing.T) {
	cfg, err := Parse("clickhouse://localhost:9000")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "localhost:9000" {
		t.Errorf("Addrs = %v, want [localhost:9000]", cfg.Addrs)
	}
	if cfg.User != "" {
		t.Errorf("User = %q, want empty", cfg.User)
	}
	if cfg.Database != "" {
		t.Errorf("Database = %q, want empty", cfg.Database)
	}
}

func TestFullAuth(t *testing.T) {
	cfg, err := Parse("clickhouse://alice:s3cret@localhost:9000/mydb")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addrs[0] != "localhost:9000" {
		t.Errorf("Addrs[0] = %q, want localhost:9000", cfg.Addrs[0])
	}
	if cfg.User != "alice" {
		t.Errorf("User = %q, want alice", cfg.User)
	}
	if cfg.Password != "s3cret" {
		t.Errorf("Password = %q, want s3cret", cfg.Password)
	}
	if cfg.Database != "mydb" {
		t.Errorf("Database = %q, want mydb", cfg.Database)
	}
}

func TestPasswordURLEncoded(t *testing.T) {
	cfg, err := Parse("clickhouse://user:p%40ss%2Fword@host:9000/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != "p@ss/word" {
		t.Errorf("Password = %q, want p@ss/word", cfg.Password)
	}
}

func TestDefaultPort(t *testing.T) {
	cfg, err := Parse("clickhouse://host")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addrs[0] != "host:9000" {
		t.Errorf("Addrs[0] = %q, want host:9000", cfg.Addrs[0])
	}
}

func TestIPv6(t *testing.T) {
	cfg, err := Parse("clickhouse://[::1]:9000/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addrs[0] != "[::1]:9000" {
		t.Errorf("Addrs[0] = %q, want [::1]:9000", cfg.Addrs[0])
	}
}

func TestConnectionParamsConsumed(t *testing.T) {
	cfg, err := Parse("clickhouse://host:9000/db?dial_timeout=10s&compress=lz4")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DialTimeout != 10*time.Second {
		t.Errorf("DialTimeout = %v, want 10s", cfg.DialTimeout)
	}
	if cfg.Compression != "lz4" {
		t.Errorf("Compression = %q, want lz4", cfg.Compression)
	}
}

func TestUnknownConnectionParam(t *testing.T) {
	cfg, err := Parse("clickhouse://host?bogus_param=1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Settings) != 1 || cfg.Settings[0].Key != "bogus_param" || cfg.Settings[0].Value != "1" {
		t.Errorf("Settings = %v, want [{bogus_param 1}]", cfg.Settings)
	}
}

func TestSettingsFromRemainingParams(t *testing.T) {
	cfg, err := Parse("clickhouse://host:9000/db?max_threads=4&allow_experimental_object_type=1")
	if err != nil {
		t.Fatal(err)
	}
	expected := []proto.Setting{
		{Key: "max_threads", Value: "4"},
		{Key: "allow_experimental_object_type", Value: "1"},
	}
	if len(cfg.Settings) != len(expected) {
		t.Fatalf("Settings length = %d, want %d", len(cfg.Settings), len(expected))
	}
	for i, s := range cfg.Settings {
		if s.Key != expected[i].Key || s.Value != expected[i].Value {
			t.Errorf("Settings[%d] = {%q, %q}, want {%q, %q}", i, s.Key, s.Value, expected[i].Key, expected[i].Value)
		}
	}
}

func TestMultiHost(t *testing.T) {
	cfg, err := Parse("clickhouse://host1:9000,host2:9001/db")
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

func TestMultiHostDefaultPort(t *testing.T) {
	cfg, err := Parse("clickhouse://host1,host2/db")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"host1:9000", "host2:9000"}
	if len(cfg.Addrs) != len(expected) {
		t.Fatalf("Addrs = %v, want %v", cfg.Addrs, expected)
	}
	for i, a := range cfg.Addrs {
		if a != expected[i] {
			t.Errorf("Addrs[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestUnknownScheme(t *testing.T) {
	_, err := Parse("http://host:9000/db")
	if err == nil {
		t.Fatal("expected error for http scheme")
	}
	var dsnErr *Error
	if e, ok := err.(*Error); ok {
		dsnErr = e
	} else {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if dsnErr.Kind != KindScheme {
		t.Errorf("error kind = %d, want KindScheme", dsnErr.Kind)
	}
}

func TestEmptyHost(t *testing.T) {
	_, err := Parse("clickhouse://:9000/db")
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestEmptyHostSegment(t *testing.T) {
	_, err := Parse("clickhouse://host1,,host2/db")
	if err == nil {
		t.Fatal("expected error for empty host segment")
	}
}

func TestSingleHost(t *testing.T) {
	cfg, err := Parse("clickhouse://host1:9000/db")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Addrs) != 1 || cfg.Addrs[0] != "host1:9000" {
		t.Errorf("Addrs = %v, want [host1:9000]", cfg.Addrs)
	}
}

func TestDurationParseError(t *testing.T) {
	_, err := Parse("clickhouse://host/db?dial_timeout=10x")
	if err == nil {
		t.Fatal("expected error for bad duration")
	}
}

func TestNoParams(t *testing.T) {
	cfg, err := Parse("clickhouse://host/db")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Settings) != 0 {
		t.Errorf("Settings = %v, want empty", cfg.Settings)
	}
}

func TestDatabasePathTraversal(t *testing.T) {
	_, err := Parse("clickhouse://host/../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal in db name")
	}
}

func TestDatabaseControlChars(t *testing.T) {
	_, err := Parse("clickhouse://host/db\x00name")
	if err == nil {
		t.Fatal("expected error for control chars in db name")
	}
}

func TestOversizedUsername(t *testing.T) {
	user := strings.Repeat("a", 1025)
	_, err := Parse("clickhouse://" + user + "@host/db")
	if err == nil {
		t.Fatal("expected error for oversized username")
	}
}

func TestOversizedPassword(t *testing.T) {
	pass := strings.Repeat("a", 1025)
	_, err := Parse("clickhouse://user:" + pass + "@host/db")
	if err == nil {
		t.Fatal("expected error for oversized password")
	}
}

func TestOversizedDatabase(t *testing.T) {
	db := strings.Repeat("a", 1025)
	_, err := Parse("clickhouse://host/" + db)
	if err == nil {
		t.Fatal("expected error for oversized database")
	}
}

func TestTooManyHosts(t *testing.T) {
	hosts := make([]string, 101)
	for i := range hosts {
		hosts[i] = "host"
	}
	dsn := "clickhouse://" + strings.Join(hosts, ",") + "/db"
	_, err := Parse(dsn)
	if err == nil {
		t.Fatal("expected error for too many hosts")
	}
}

func TestSanitizeForError(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "hello"},
		{"pass\nword", "pass.word"},
		{"pass\rword", "pass.word"},
		{"pass\x00word", "pass.word"},
		{"normal", "normal"},
	}
	for _, tt := range tests {
		got := SanitizeForError(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeForError(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCompressionMapping(t *testing.T) {
	// Empty = no compression
	cfg, err := Parse("clickhouse://host/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Compression != "" {
		t.Errorf("default Compression = %q, want empty", cfg.Compression)
	}
}
