package config

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

const (
	DefaultSMTPPort       = 1026
	DefaultHTTPPort       = 8026
	DefaultMaxMessages    = 1000
	DefaultMaxMessageSize = 10 << 20
	DefaultMaxStoreSize   = 256 << 20
)

type Config struct {
	Host           string
	SMTPPort       int
	HTTPPort       int
	MaxMessages    int
	MaxMessageSize int64
	MaxStoreSize   int64
	SMTPAuth       Credentials
	UIAuth         Credentials
	SMTPTLS        bool
	SMTPTLSCert    string
	SMTPTLSKey     string
	DataDir        string
}

type Credentials struct {
	User     string
	Password string
}

func (c Credentials) Enabled() bool { return c.User != "" }

func (c Credentials) Match(user, password string) bool {
	hash := func(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }
	userOK := subtle.ConstantTimeCompare(hash(user), hash(c.User))
	passOK := subtle.ConstantTimeCompare(hash(password), hash(c.Password))
	return c.Enabled() && userOK&passOK == 1
}

func ParseCredentials(s string) (Credentials, error) {
	user, password, ok := strings.Cut(s, ":")
	if !ok || user == "" {
		return Credentials{}, errors.New(`credentials must look like "user:password"`)
	}
	return Credentials{User: user, Password: password}, nil
}

func (c Config) SMTPAddr() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.SMTPPort)) }

func (c Config) HTTPAddr() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.HTTPPort)) }

var ErrHelp = flag.ErrHelp

func Load(args []string, getenv func(string) string, output io.Writer) (Config, error) {
	cfg := Config{
		Host:           "localhost",
		SMTPPort:       DefaultSMTPPort,
		HTTPPort:       DefaultHTTPPort,
		MaxMessages:    DefaultMaxMessages,
		MaxMessageSize: DefaultMaxMessageSize,
		MaxStoreSize:   DefaultMaxStoreSize,
	}
	if err := applyEnv(&cfg, getenv); err != nil {
		return Config{}, err
	}

	fs := flag.NewFlagSet("mailpeek", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Host, "host", cfg.Host, "interface to bind: localhost is this machine only; 0.0.0.0 accepts other machines and containers (env MAILPEEK_HOST)")
	fs.IntVar(&cfg.SMTPPort, "smtp-port", cfg.SMTPPort, "SMTP port (env MAILPEEK_SMTP_PORT)")
	fs.IntVar(&cfg.HTTPPort, "http-port", cfg.HTTPPort, "HTTP port for Web UI and API (env MAILPEEK_HTTP_PORT)")
	fs.IntVar(&cfg.MaxMessages, "max-messages", cfg.MaxMessages, "messages kept in memory; oldest are removed (env MAILPEEK_MAX_MESSAGES)")
	size := sizeFlag{&cfg.MaxMessageSize}
	fs.Var(size, "max-message-size", "maximum message size, e.g. 10MB (env MAILPEEK_MAX_MESSAGE_SIZE)")
	fs.Var(sizeFlag{&cfg.MaxStoreSize}, "max-store-size", "memory for all stored messages; oldest are removed first (env MAILPEEK_MAX_STORE_SIZE)")
	fs.Var(credFlag{&cfg.SMTPAuth}, "smtp-auth", "require SMTP login, as user:password (env MAILPEEK_SMTP_AUTH)")
	fs.Var(credFlag{&cfg.UIAuth}, "ui-auth", "protect the Web UI and API with HTTP Basic auth, as user:password (env MAILPEEK_UI_AUTH)")
	fs.BoolVar(&cfg.SMTPTLS, "smtp-tls", cfg.SMTPTLS, "offer STARTTLS on SMTP, with a self-signed certificate unless --smtp-tls-cert is given (env MAILPEEK_SMTP_TLS)")
	fs.StringVar(&cfg.SMTPTLSCert, "smtp-tls-cert", cfg.SMTPTLSCert, "PEM certificate for STARTTLS; implies --smtp-tls (env MAILPEEK_SMTP_TLS_CERT)")
	fs.StringVar(&cfg.SMTPTLSKey, "smtp-tls-key", cfg.SMTPTLSKey, "PEM private key for --smtp-tls-cert (env MAILPEEK_SMTP_TLS_KEY)")
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "keep messages in this directory so they survive restarts; off keeps them in memory only (env MAILPEEK_DATA_DIR)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if cfg.SMTPTLSCert != "" {
		cfg.SMTPTLS = true
	}
	return cfg, cfg.validate()
}

func applyEnv(cfg *Config, getenv func(string) string) error {
	if v := getenv("MAILPEEK_HOST"); v != "" {
		cfg.Host = v
	}
	if v := getenv("MAILPEEK_SMTP_TLS"); v != "" {
		on, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("MAILPEEK_SMTP_TLS: use true or false, got %q", v)
		}
		cfg.SMTPTLS = on
	}
	cfg.SMTPTLSCert = orEnv(getenv("MAILPEEK_SMTP_TLS_CERT"), cfg.SMTPTLSCert)
	cfg.SMTPTLSKey = orEnv(getenv("MAILPEEK_SMTP_TLS_KEY"), cfg.SMTPTLSKey)
	cfg.DataDir = orEnv(getenv("MAILPEEK_DATA_DIR"), cfg.DataDir)
	ints := []struct {
		key string
		dst *int
	}{
		{"MAILPEEK_SMTP_PORT", &cfg.SMTPPort},
		{"MAILPEEK_HTTP_PORT", &cfg.HTTPPort},
		{"MAILPEEK_MAX_MESSAGES", &cfg.MaxMessages},
	}
	for _, e := range ints {
		v := getenv(e.key)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("%s: invalid number %q", e.key, v)
		}
		*e.dst = n
	}
	sizes := []struct {
		key string
		dst *int64
	}{{"MAILPEEK_MAX_MESSAGE_SIZE", &cfg.MaxMessageSize}, {"MAILPEEK_MAX_STORE_SIZE", &cfg.MaxStoreSize}}
	for _, e := range sizes {
		if v := getenv(e.key); v != "" {
			n, err := ParseSize(v)
			if err != nil {
				return fmt.Errorf("%s: %w", e.key, err)
			}
			*e.dst = n
		}
	}
	creds := []struct {
		key string
		dst *Credentials
	}{{"MAILPEEK_SMTP_AUTH", &cfg.SMTPAuth}, {"MAILPEEK_UI_AUTH", &cfg.UIAuth}}
	for _, e := range creds {
		if v := getenv(e.key); v != "" {
			c, err := ParseCredentials(v)
			if err != nil {
				return fmt.Errorf("%s: %w", e.key, err)
			}
			*e.dst = c
		}
	}
	return nil
}

func (c Config) validate() error {
	var errs []error
	for name, port := range map[string]int{"smtp-port": c.SMTPPort, "http-port": c.HTTPPort} {
		if port < 0 || port > 65535 {
			errs = append(errs, fmt.Errorf("%s: %d is not a valid port", name, port))
		}
	}
	if c.SMTPPort != 0 && c.SMTPPort == c.HTTPPort {
		errs = append(errs, errors.New("smtp-port and http-port must differ"))
	}
	if c.MaxMessages < 1 {
		errs = append(errs, errors.New("max-messages must be at least 1"))
	}
	if c.MaxMessageSize < 1 {
		errs = append(errs, errors.New("max-message-size must be positive"))
	}
	if c.MaxStoreSize < c.MaxMessageSize {
		errs = append(errs, errors.New("max-store-size must be at least max-message-size"))
	}
	if (c.SMTPTLSCert == "") != (c.SMTPTLSKey == "") {
		errs = append(errs, errors.New("smtp-tls-cert and smtp-tls-key must be given together"))
	}
	return errors.Join(errs...)
}

func orEnv(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

func ParseSize(s string) (int64, error) {
	v := strings.ToUpper(strings.TrimSpace(s))
	units := []struct {
		suffix string
		mult   int64
	}{{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"G", 1 << 30}, {"M", 1 << 20}, {"K", 1 << 10}, {"B", 1}}
	mult := int64(1)
	for _, u := range units {
		if strings.HasSuffix(v, u.suffix) {
			v, mult = strings.TrimSpace(strings.TrimSuffix(v, u.suffix)), u.mult
			break
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n * mult, nil
}

func FormatSize(n int64) string {
	switch {
	case n >= 1<<30 && n%(1<<30) == 0:
		return fmt.Sprintf("%dGB", n>>30)
	case n >= 1<<20 && n%(1<<20) == 0:
		return fmt.Sprintf("%dMB", n>>20)
	case n >= 1<<10 && n%(1<<10) == 0:
		return fmt.Sprintf("%dKB", n>>10)
	}
	return fmt.Sprintf("%dB", n)
}

type sizeFlag struct{ dst *int64 }

func (f sizeFlag) String() string {
	if f.dst == nil {
		return ""
	}
	return FormatSize(*f.dst)
}

func (f sizeFlag) Set(s string) error {
	n, err := ParseSize(s)
	if err != nil {
		return err
	}
	*f.dst = n
	return nil
}

type credFlag struct{ dst *Credentials }

func (f credFlag) String() string { return "" }

func (f credFlag) Set(s string) error {
	c, err := ParseCredentials(s)
	if err != nil {
		return err
	}
	*f.dst = c
	return nil
}
