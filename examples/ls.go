package main

import (
	"log"

	"github.com/jbain/ezec"
	"github.com/jbain/ezec/pkg/config"
	"github.com/jbain/ezec/pkg/consumers"
)

func main() {
	lscmd := ezec.Command("ls", config.StringArgs{
		String: "-alh ../",
	})

	logParser := consumers.NewStdoutLogger("stdout", 100)
	go logParser.Start()

	lscmd.Stdout = []ezec.LineConsumer{
		logParser,
	}

	err := lscmd.Start()
	if err != nil {
		log.Printf("error starting cmd: %s", err.Error())
	}

	if err := lscmd.Wait(); err != nil {
		log.Printf("error waiting for cmd: %s", err.Error())
	}
	logParser.Wait()
}
