package utils

import (
	"container/list"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func ConfigureLogger(level string) {
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

func PrintBlockchain(blockchain *list.List) {

}

func PrintLog(log *list.List) {

}
