package config

import (
	"io"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(nil, env(nil), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != 1026 || cfg.HTTPPort != 8026 || cfg.MaxMessages != 100 || cfg.MaxMessageSize != 10<<20 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.SMTPAddr() != "0.0.0.0:1026" || cfg.HTTPAddr() != "0.0.0.0:8026" {
		t.Fatalf("unexpected addrs: %s %s", cfg.SMTPAddr(), cfg.HTTPAddr())
	}
}

func TestLoadEnv(t *testing.T) {
	cfg, err := Load(nil, env(map[string]string{
		"MAILPEEK_SMTP_PORT":        "2525",
		"MAILPEEK_HTTP_PORT":        "9000",
		"MAILPEEK_MAX_MESSAGES":     "5",
		"MAILPEEK_MAX_MESSAGE_SIZE": "1MB",
	}), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != 2525 || cfg.HTTPPort != 9000 || cfg.MaxMessages != 5 || cfg.MaxMessageSize != 1<<20 {
		t.Fatalf("env not applied: %+v", cfg)
	}
}

func TestFlagsOverrideEnv(t *testing.T) {
	cfg, err := Load(
		[]string{"--smtp-port", "3025", "--max-messages", "7", "--max-message-size", "512KB"},
		env(map[string]string{"MAILPEEK_SMTP_PORT": "2525", "MAILPEEK_MAX_MESSAGES": "5"}),
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != 3025 || cfg.MaxMessages != 7 || cfg.MaxMessageSize != 512<<10 {
		t.Fatalf("flags did not take precedence: %+v", cfg)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{"bad env number", nil, map[string]string{"MAILPEEK_HTTP_PORT": "abc"}},
		{"bad env size", nil, map[string]string{"MAILPEEK_MAX_MESSAGE_SIZE": "lots"}},
		{"port out of range", []string{"--smtp-port", "70000"}, nil},
		{"same ports", []string{"--smtp-port", "8026"}, nil},
		{"zero messages", []string{"--max-messages", "0"}, nil},
		{"positional arg", []string{"serve"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Load(tc.args, env(tc.env), io.Discard); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"10MB": 10 << 20, "10mb": 10 << 20, "512KB": 512 << 10, "1G": 1 << 30,
		"2048": 2048, "100B": 100, " 3 MB ": 3 << 20, //nolint:gocritic
	}
	for in, want := range cases {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "MB", "-1", "1.5MB", "ten"} {
		if _, err := ParseSize(in); err == nil {
			t.Errorf("ParseSize(%q) expected error", in)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{10 << 20: "10MB", 512 << 10: "512KB", 1 << 30: "1GB", 1000: "1000B"}
	for in, want := range cases {
		if got := FormatSize(in); got != want {
			t.Errorf("FormatSize(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestCredentials(t *testing.T) {
	cfg, err := Load(
		[]string{"--ui-auth", "admin:pa:ss"},
		env(map[string]string{"MAILPEEK_SMTP_AUTH": "app:secret", "MAILPEEK_UI_AUTH": "ignored:x"}),
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPAuth != (Credentials{"app", "secret"}) || cfg.UIAuth != (Credentials{"admin", "pa:ss"}) {
		t.Fatalf("credentials = %+v %+v", cfg.SMTPAuth, cfg.UIAuth)
	}
	if !cfg.UIAuth.Match("admin", "pa:ss") || cfg.UIAuth.Match("admin", "pa") || cfg.UIAuth.Match("Admin", "pa:ss") {
		t.Fatal("Match is wrong")
	}
	if (Credentials{}).Enabled() || (Credentials{}).Match("", "") {
		t.Fatal("empty credentials must never match")
	}
	for _, bad := range []string{"nocolon", ":password", ""} {
		if _, err := ParseCredentials(bad); err == nil {
			t.Errorf("ParseCredentials(%q) should fail", bad)
		}
	}
	if _, err := Load([]string{"--smtp-auth", "bad"}, env(nil), io.Discard); err == nil {
		t.Fatal("expected flag error")
	}
	if _, err := Load(nil, env(map[string]string{"MAILPEEK_UI_AUTH": "bad"}), io.Discard); err == nil {
		t.Fatal("expected env error")
	}
}

func TestMaxStoreSize(t *testing.T) {
	cfg, err := Load([]string{"--max-store-size", "1GB"}, env(nil), io.Discard)
	if err != nil || cfg.MaxStoreSize != 1<<30 {
		t.Fatalf("flag: %v %d", err, cfg.MaxStoreSize)
	}
	cfg, err = Load(nil, env(map[string]string{"MAILPEEK_MAX_STORE_SIZE": "64MB"}), io.Discard)
	if err != nil || cfg.MaxStoreSize != 64<<20 {
		t.Fatalf("env: %v %d", err, cfg.MaxStoreSize)
	}
	if def, _ := Load(nil, env(nil), io.Discard); def.MaxStoreSize != DefaultMaxStoreSize {
		t.Fatalf("default = %d", def.MaxStoreSize)
	}
	if _, err := Load([]string{"--max-store-size", "1MB"}, env(nil), io.Discard); err == nil {
		t.Fatal("store smaller than one message must be rejected")
	}
}
