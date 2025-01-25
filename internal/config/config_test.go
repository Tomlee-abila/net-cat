package config

import (
"testing"
"time"
)

func TestDefaultConfig(t *testing.T) {
cfg := DefaultConfig()

if cfg == nil {
t.Fatal("DefaultConfig() returned nil")
}

// Test default values
tests := []struct {
name     string
got      interface{}
want     interface{}
}{
{
name: "Default listen address",
got:  cfg.ListenAddr,
want: ":8989",
},
{
name: "Max clients",
got:  cfg.MaxClients,
want: 10,
},
{
name: "Max name length",
got:  cfg.MaxNameLength,
want: 32,
},
{
name: "Max message size",
got:  cfg.MaxMessageSize,
want: 1024,
},
{
name: "Client timeout",
got:  cfg.ClientTimeout,
want: 5 * time.Minute,
},
{
name: "Message rate limit",
got:  cfg.MessageRateLimit,
want: time.Second,
},
{
name: "Max name changes",
got:  cfg.MaxNameChanges,
want: 3,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if tt.got != tt.want {
t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
}
})
}
}

func TestWithListenAddr(t *testing.T) {
tests := []struct {
name    string
addr    string
want    string
}{
{
name: "Custom port",
addr: ":2525",
want: ":2525",
},
{
name: "Default port",
addr: ":8989",
want: ":8989",
},
{
name: "Custom host and port",
addr: "localhost:8080",
want: "localhost:8080",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
cfg := DefaultConfig()
cfg = cfg.WithListenAddr(tt.addr)

if cfg.ListenAddr != tt.want {
t.Errorf("ListenAddr = %s, want %s", cfg.ListenAddr, tt.want)
}
})
}
}

func TestConfigBuilder(t *testing.T) {
customTimeout := 2 * time.Minute
customRateLimit := 500 * time.Millisecond

cfg := DefaultConfig().
WithListenAddr(":2525").
WithMaxClients(20).
WithMaxNameLength(30).
WithMaxMessageSize(2048).
WithClientTimeout(customTimeout).
WithMessageRateLimit(customRateLimit).
WithMaxNameChanges(5).
WithLogFile("/var/log/chat.log")

tests := []struct {
name string
got  interface{}
want interface{}
}{
{"ListenAddr", cfg.ListenAddr, ":2525"},
{"MaxClients", cfg.MaxClients, 20},
{"MaxNameLength", cfg.MaxNameLength, 30},
{"MaxMessageSize", cfg.MaxMessageSize, 2048},
{"ClientTimeout", cfg.ClientTimeout, customTimeout},
{"MessageRateLimit", cfg.MessageRateLimit, customRateLimit},
{"MaxNameChanges", cfg.MaxNameChanges, 5},
{"LogFile", cfg.LogFile, "/var/log/chat.log"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if tt.got != tt.want {
t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
}
})
}

// Test method chaining
newCfg := cfg.
WithMaxClients(30).
WithMaxNameLength(40)

if newCfg != cfg {
t.Error("Method chaining should return same config instance")
}
}

func TestConfigImmutability(t *testing.T) {
original := DefaultConfig()
originalAddr := original.ListenAddr

modified := original.WithListenAddr(":9999")

if original != modified {
t.Error("WithListenAddr should modify and return same instance")
}

if originalAddr == modified.ListenAddr {
t.Error("ListenAddr should be modified")
}
}

func TestLogFileConfig(t *testing.T) {
tests := []struct {
name     string
logFile  string
wantFile string
}{
{
name:     "Empty log file",
logFile:  "",
wantFile: "",
},
{
name:     "Valid log file path",
logFile:  "/var/log/chat.log",
wantFile: "/var/log/chat.log",
},
{
name:     "Relative log file path",
logFile:  "chat.log",
wantFile: "chat.log",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
cfg := DefaultConfig().WithLogFile(tt.logFile)

if cfg.LogFile != tt.wantFile {
t.Errorf("LogFile = %v, want %v", cfg.LogFile, tt.wantFile)
}
})
}
}
