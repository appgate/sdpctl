package main

import (
	"os"
	"strings"

	"github.com/appgate/appgatectl/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Setup logging
	logLevel := strings.ToLower(os.Getenv("APPGATECTL_LOG_LEVEL"))

	switch logLevel {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}

	f := log.TextFormatter{
		FullTimestamp: true,
		PadLevelText: true,
	}
	log.SetFormatter(&f)
}

func main() {
	cmd.Execute()
}
