//go:build gui

package main

import (
	"flag"
	"log"

	"kefu-server/appboot"
	"kefu-server/systray"
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

	openURL := appboot.BuildOpenURL(cfg.Admin.Address)
	log.Printf("[server] gui mode on, listen=%s open=%s", cfg.Admin.Address, openURL)

	systray.Init("零点客服", "零点客服服务运行中", openURL, nil)
	systray.Run(func() {
		go func() {
			if err := r.Run(cfg.Admin.Address); err != nil {
				log.Fatal(err)
			}
		}()
	})
}
