package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"gioui.org/app"

	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui"
	_ "github.com/planetdecred/godcr/ui/assets"
	"github.com/planetdecred/godcr/wallet"
)

var (
	Version   string = "0.0.0"
	BuildDate string
	BuildEnv  = wallet.DevBuild
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	if cfg.Profile > 0 {
		go func() {
			log.Info(fmt.Sprintf("Starting profiling server on port %d", cfg.Profile))
			log.Error(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", cfg.Profile), nil))
		}()
	}

	dcrlibwallet.SetLogLevels(cfg.DebugLevel)

	var buildDate time.Time
	if BuildEnv == wallet.ProdBuild {
		buildDate, err = time.Parse(time.RFC3339, BuildDate)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return
		}
	} else {
		buildDate = time.Now()
	}

	var net string
	switch cfg.Network {
	case "testnet":
		net = "testnet3"
	default:
		net = cfg.Network
	}

	logFile := filepath.Join(cfg.LogDir, defaultLogFilename)
	wal, err := wallet.NewWallet(cfg.HomeDir, net, Version, logFile, buildDate, make(chan wallet.Response, 3))
	if err != nil {
		log.Error(err)
		return
	}

	shutdown := make(chan int)
	go func() {
		<-shutdown
		wal.Shutdown()
		os.Exit(0)
	}()

	win, appWindow, err := ui.CreateWindow(wal)
	if err != nil {
		log.Errorf("Could not initialize window: %s\ns", err)
		return
	}

	// Start the ui frontend
	go win.Loop(appWindow, shutdown)

	app.Main()
}
