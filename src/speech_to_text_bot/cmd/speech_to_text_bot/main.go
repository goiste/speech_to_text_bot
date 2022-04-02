package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"speech_to_text_bot/handlers"
	"speech_to_text_bot/yandex"

	log "github.com/sirupsen/logrus"
)

func main() {
	config, err := readConfig()
	if err != nil {
		log.WithError(err).Error("Read config error")
		return
	}

	awsConfig := &yandex.ClientConfig{
		AwsRegion:          config.AwsRegion,
		AwsStaticKeyID:     config.AwsStaticKeyID,
		AwsStaticKeySecret: config.AwsStaticKeySecret,
		AwsApiKey:          config.AwsApiKey,
		AwsBucket:          config.AwsBucket,
	}

	ycl, err := yandex.New(awsConfig)
	if err != nil {
		log.WithError(err).Error("get yandex client error")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	botHandler := handlers.New(ycl, config.BotToken, config.BotOwnerID, config.TrustedIDs)

	errChan := make(chan error, 1)
	defer close(errChan)

	go botHandler.Start(ctx, errChan)

	c := make(chan os.Signal)
	defer close(c)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-c:
			return
		case e := <-errChan:
			_ = botHandler.SendLog(handlers.LogLevelError, e.Error())
			log.Errorln(e)
		}
	}
}
