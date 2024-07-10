package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/xueqianLu/ethtools/cmd"
)

func main() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02 15:04:05.000"})

	cmd.Execute()
}
