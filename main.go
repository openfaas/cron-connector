// Copyright (c) Alex Ellis 2017. All rights reserved.
// Copyright (c) OpenFaaS Project 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"os"
	"time"

	"github.com/openfaas-incubator/connector-sdk/types"
)

func main() {
	creds := types.GetCredentials()
	config, err := getControllerConfig()
	if err != nil {
		panic(err)
	}

	controller := types.NewController(creds, config)
	controller.BeginMapBuilder()

	topic := "faas-request"
	invokeTime := time.Second * 2

	// Simulate events emitting from queue/pub-sub
	for {
		time.Sleep(invokeTime)
		data := []byte("test " + time.Now().String())

		controller.Invoke(topic, &data)
	}
}

func getControllerConfig() (*types.ControllerConfig, error) {
	gURL, ok := os.LookupEnv("gateway_url")
	if !ok {
		return nil, errors.New("Gateway URL not set")
	}

	return &types.ControllerConfig{
		RebuildInterval: time.Millisecond * 1000,
		GatewayURL:      gURL,
		PrintResponse:   true,
	}, nil
}
