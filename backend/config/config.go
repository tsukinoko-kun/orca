package config

import "os"

var (
	Address = "0.0.0.0:8080"
	Home, _ = os.UserHomeDir()
)
