package validation

import "testing"

func TestValidatePort(t *testing.T) {
tests := []struct {
name    string
port    string
wantErr bool
}{
{
name:    "Valid port",
port:    "8080",
wantErr: false,
},
{
name:    "Valid port with brackets",
port:    "[8080]",
wantErr: false,
},
{
name:    "Port zero",
port:    "0",
wantErr: true,
},
{
name:    "Negative port",
port:    "-1",
wantErr: true,
},
{
name:    "Port too high",
port:    "65536",
wantErr: true,
},
{
name:    "Invalid port string",
port:    "abc",
wantErr: true,
},
{
name:    "Empty port",
port:    "",
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := ValidatePort(tt.port)
if (err != nil) != tt.wantErr {
t.Errorf("ValidatePort() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestIsValidPort(t *testing.T) {
tests := []struct {
name string
port string
want bool
}{
{
name: "Valid port",
port: "8080",
want: true,
},
{
name: "Invalid port",
port: "abc",
want: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if got := IsValidPort(tt.port); got != tt.want {
t.Errorf("IsValidPort() = %v, want %v", got, tt.want)
}
})
}
}

func TestNormalizePort(t *testing.T) {
tests := []struct {
name string
port string
want string
}{
{
name: "Port with brackets",
port: "[8080]",
want: "8080",
},
{
name: "Port without brackets",
port: "8080",
want: "8080",
},
{
name: "Empty port",
port: "",
want: "",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
if got := NormalizePort(tt.port); got != tt.want {
t.Errorf("NormalizePort() = %v, want %v", got, tt.want)
}
})
}
}
