package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/op/go-logging"
)

func initLogging() {
	LOG = logging.MustGetLogger("mk-mq-d")
	formatter := logging.MustStringFormatter(logFormat)
	lb := logging.NewLogBackend(os.Stdout, "", 0)
	lbf := logging.NewBackendFormatter(lb, formatter)
	lbl := logging.AddModuleLevel(lbf)
	logging.SetBackend(lbl)
}

func initInterrupt() {
	LOG.Info("-- start --")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(c chan os.Signal) {
		for range c {
			LOG.Info("-- stop --")
			os.Exit(137)
		}
	}(c)
}

func initReadConf() {
	fCfgPath := flag.String("c", "config.toml", "path to conf file")
	flag.Parse()

	LOG.Debug("read config from " + *fCfgPath)
	CFG = LoadConfig(*fCfgPath)
}

func init() {
	initLogging()
	initInterrupt()
	initReadConf()
}
