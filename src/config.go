package main

import (
	"os"
)

// PORT value read from environment variable or default 3000
var PORT = getPort()

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return port
}
