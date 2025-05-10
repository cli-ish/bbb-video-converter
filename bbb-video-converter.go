package main

import (
	"fmt"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"github.com/cli-ish/bbb-video-converter/internal/converter"
	"log"
	"os"
	"time"
)

func main() {
	configData := config.Data{}
	err := configData.LoadConfig()
	if err != nil {
		// Something with the configuration did not work, lets exit here.
		fmt.Println(err)
		os.Exit(1)
	}
	log.Println("Starting the conversion")
	log.Println("========================================================")
	log.Println("Recording:\t" + configData.RecordingDir)
	log.Println("Output:\t" + configData.OutputFile)
	log.Println("========================================================")
	tool := converter.Converter{}
	startTime := time.Now()
	err = tool.Run(configData)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	duration := time.Now().Sub(startTime)
	log.Println("Finished, took:" + duration.String())
}
