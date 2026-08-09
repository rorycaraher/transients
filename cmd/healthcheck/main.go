// Command healthcheck is a Docker HEALTHCHECK probe: it has no shell or
// curl to exec inside the scratch image, so this GETs /healthz itself and
// exits non-zero on anything but 200.
package main

import (
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
