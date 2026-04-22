//go:build !gui

package main

import (
	"flag"
	"log"

	"kefu-server/appboot"
)

func main() {
	addr := flag.String("addr", "", "Server listen address, e.g. 0.0.0.0:5300")
	dataDir := flag.String("data", "", "Data directory path. Default: %APPDATA%/kefu (Windows), ~/Library/Application Support/kefu (macOS), ~/.kefu (Linux)")
	logLevel := flag.String("log-level", "", "Log level: trace|debug|info|warn|error|fatal|panic")
	jwtSecret := flag.String("jwt-secret", "", "JWT signing secret (optional)")
	flag.Parse()

	r, cfg, err := appboot.InitRuntime(appboot.Options{
		Addr:      *addr,
		DataDir:   *dataDir,
		JWTSecret: *jwtSecret,
		LogLevel:  *logLevel,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("[server] starting on %s", cfg.Admin.Address)
	if err := r.Run(cfg.Admin.Address); err != nil {
		log.Fatal(err)
	}
}
