package main

import (
	"PAXOS-Banking/server/consensus"
	"PAXOS-Banking/utils"
	"context"
	"flag"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
)

func main() {

	var (
		logLevel string
		id       int
	)
	_, cancel := context.WithCancel(context.Background())
	// parse all the command line arguments
	flag.StringVar(&logLevel, "level", "DEBUG", "Set log level.")
	flag.IntVar(&id, "id", 1, "id of the server(1,2,3)")
	flag.Parse()

	utils.ConfigureLogger(logLevel)

	consensus.Start(id)

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			log.Info("Received an interrupt, stopping all connections...")
			consensus.ClearRedisData()
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}
