// entry point to app :)
package main

import (
	"fmt"

	"github.com/ds124wfegd/WB_L3/2/config"
	"github.com/ds124wfegd/WB_L3/2/internal/appServer"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))

	viperInstance, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Cannot load config. Error: {%s}", err.Error())
	}

	cfg, err := config.ParseConfig(viperInstance)
	if err != nil {
		logrus.Fatalf("Cannot parse config. Error: {%s}", err.Error())
	}

	fmt.Println(cfg)
	appServer.NewServer(cfg)
}
