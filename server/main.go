package main

import (
	"PAXOS-Banking/server/consensus"
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func configureLogger(level string) {
	log.SetOutput(os.Stderr)
	switch strings.ToLower(level) {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning", "warn":
		log.SetLevel(log.WarnLevel)
	}
}

func main() {

	var (
		logLevel   string
		id int
	)
	_, cancel := context.WithCancel(context.Background())
	// parse all the command line arguments
	flag.StringVar(&logLevel, "level", "DEBUG", "Set log level.")
	flag.IntVar(&id, "id", 1, "id of the server(1,2,3)")
	flag.Parse()

	configureLogger(logLevel)

	consensus.Start(id)

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			log.Info("Received an interrupt, stopping all connections...")
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}
