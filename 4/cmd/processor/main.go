package main

import (
	"github.com/ds124wfegd/WB_L3/4/config"
	"github.com/ds124wfegd/WB_L3/4/internal/pkg/processor"
)

func main() {
	processor.StartImageProcessorConsumer(
		[]string{config.GetEnv("KAFKA_BROKERS", "localhost:9094")},
		config.GetEnv("KAFKA_TOPIC", "images"),
		config.GetEnv("KAFKA_GROUP_ID", "image-processor-service"),
	)
}
