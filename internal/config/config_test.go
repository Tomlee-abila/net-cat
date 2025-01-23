package config

import (
"testing"
"time"
)

func TestDefaultConfig(t *testing.T) {
cfg := DefaultConfig()

tests := []struct {
name     string
got      interface{}
want     interface{}
checkFun func(got, want interface{}) bool
}{
{
name: "Default ListenAddr",
got:  cfg.ListenAddr,
want: ":8989",
checkFun: func(got, want interface{}) bool {
return got.(string) == want.(string)
},
},
{
name: "Default MaxClients",
got:  cfg.MaxClients,
want: 10,
checkFun: func(got, want interface{}) bool {
return got.(int) == want.(int)
},
},
{
name: "Default ClientTimeout",
got:  cfg.ClientTimeout,
want: 5 * time.Minute,
checkFun: func(got, want interface{}) bool {
return got.(time.Duration) == want.(time.Duration)
},
},
{
name: "Default MessageRateLimit",
got:  cfg.MessageRateLimit,
want: time.Second,
checkFun: func(got, want interface{}) bool {
return got.(time.Duration) == want.(time.Duration)
},
},
{
name: "Default MaxMessageSize",
got:  cfg.MaxMessageSize,
want: 1024,
checkFun: func(got, want interface{}) bool {
return got.(int) == want.(int)
},
},
{
name: "Default MaxNameLength",
got:  cfg.MaxNameLength,
want: 32,
checkFun: func(got, want interface{}) bool {
return got.(int) == want.(int)
},
},
{
name: "Default MaxNameChanges",
got:  cfg.MaxNameChanges,
want: 3,
checkFun: func(got, want interface{}) bool {
return got.(int) == want.(int)
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if !tt.checkFun(tt.got, tt.want) {
t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
}
})
}
}

func TestConfigChaining(t *testing.T) {
cfg := DefaultConfig().
WithListenAddr(":9000").
WithMaxClients(20).
WithClientTimeout(10*time.Minute).
WithMessageRateLimit(2*time.Second).
WithMaxMessageSize(2048).
WithMaxNameLength(64).
WithMaxNameChanges(5)

tests := []struct {
name string
got  interface{}
want interface{}
}{
{"Custom ListenAddr", cfg.ListenAddr, ":9000"},
{"Custom MaxClients", cfg.MaxClients, 20},
{"Custom ClientTimeout", cfg.ClientTimeout, 10 * time.Minute},
{"Custom MessageRateLimit", cfg.MessageRateLimit, 2 * time.Second},
{"Custom MaxMessageSize", cfg.MaxMessageSize, 2048},
{"Custom MaxNameLength", cfg.MaxNameLength, 64},
{"Custom MaxNameChanges", cfg.MaxNameChanges, 5},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if tt.got != tt.want {
t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
}
})
}
}

func TestConfigValidation(t *testing.T) {
tests := []struct {
name   string
modify func(*Config)
valid  bool
}{
{
name: "Valid default config",
modify: func(c *Config) {},
valid: true,
},
{
name: "Invalid max clients",
modify: func(c *Config) { c.MaxClients = -1 },
valid: false,
},
{
name: "Invalid client timeout",
modify: func(c *Config) { c.ClientTimeout = -time.Second },
valid: false,
},
{
name: "Invalid message rate limit",
modify: func(c *Config) { c.MessageRateLimit = 0 },
valid: false,
},
{
name: "Invalid max message size",
modify: func(c *Config) { c.MaxMessageSize = 0 },
valid: false,
},
{
name: "Invalid max name length",
modify: func(c *Config) { c.MaxNameLength = 0 },
valid: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
cfg := DefaultConfig()
tt.modify(cfg)

isValid := cfg.MaxClients > 0 &&
cfg.ClientTimeout > 0 &&
cfg.MessageRateLimit > 0 &&
cfg.MaxMessageSize > 0 &&
cfg.MaxNameLength > 0

if isValid != tt.valid {
t.Errorf("Config validation = %v, want %v", isValid, tt.valid)
}
})
}
}
