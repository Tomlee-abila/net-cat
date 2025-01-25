package config

import "time"

// Config holds all server configuration parameters
type Config struct {
	// Network settings
	ListenAddr string
	MaxClients int

	// Connection settings
	ClientTimeout    time.Duration
	MessageRateLimit time.Duration
	MaxMessageSize   int

	// Chat settings
	MaxNameLength int
	MaxNameChanges int

	// Logging settings
	LogFile string
}

// DefaultConfig returns a new Config instance with default values
func DefaultConfig() *Config {
return &Config{
ListenAddr:       ":8989",
MaxClients:       10,
ClientTimeout:    time.Minute * 5,
MessageRateLimit: time.Second,
MaxMessageSize:   1024,
MaxNameLength:    32,
MaxNameChanges:   3,
}
}

// WithListenAddr sets the listen address and returns the config
func (c *Config) WithListenAddr(addr string) *Config {
c.ListenAddr = addr
return c
}

// WithMaxClients sets the maximum number of clients and returns the config
func (c *Config) WithMaxClients(max int) *Config {
c.MaxClients = max
return c
}

// WithClientTimeout sets the client timeout duration and returns the config
func (c *Config) WithClientTimeout(timeout time.Duration) *Config {
c.ClientTimeout = timeout
return c
}

// WithMessageRateLimit sets the message rate limit and returns the config
func (c *Config) WithMessageRateLimit(limit time.Duration) *Config {
c.MessageRateLimit = limit
return c
}

// WithMaxMessageSize sets the maximum message size and returns the config
func (c *Config) WithMaxMessageSize(size int) *Config {
c.MaxMessageSize = size
return c
}

// WithMaxNameLength sets the maximum username length and returns the config
func (c *Config) WithMaxNameLength(length int) *Config {
c.MaxNameLength = length
return c
}

// WithMaxNameChanges sets the maximum number of name changes and returns the config
func (c *Config) WithMaxNameChanges(changes int) *Config {
c.MaxNameChanges = changes
return c
}

// WithLogFile sets the log file path and returns the config
func (c *Config) WithLogFile(path string) *Config {
    c.LogFile = path
    return c
}
