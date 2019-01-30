// Copyright (c) Alex Ellis 2017. All rights reserved.
// Copyright (c) OpenFaaS Project 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/openfaas-incubator/connector-sdk/types"
	cfunction "github.com/zeerorg/cron-connector/types"
)

func main() {
	creds := types.GetCredentials()
	config, err := getControllerConfig()

	if err != nil {
		panic(err)
	}

	controller := types.NewController(creds, config)
	cronScheduler := cfunction.NewScheduler()
	topic := "cron-function"
	interval := time.Second * 10

	cronScheduler.Start()
	err = startFunctionProbe(interval, topic, controller, cronScheduler, controller.Invoker)

	if err != nil {
		panic(err)
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

func startFunctionProbe(interval time.Duration, topic string, c *types.Controller, cronScheduler *cfunction.Scheduler, invoker *types.Invoker) error {
	cFs := make(cfunction.CronFunctions, 0)
	lookupBuilder := cfunction.FunctionLookupBuilder{
		GatewayURL:  c.Config.GatewayURL,
		Client:      types.MakeClient(c.Config.UpstreamTimeout),
		Credentials: c.Credentials,
	}
	ticker := time.NewTicker(interval)
	jobEntries := make(map[string]cfunction.EntryID)

	defer ticker.Stop()

	for {
		<-ticker.C
		functions, err := lookupBuilder.GetFunctions()

		if err != nil {
			return errors.New(fmt.Sprint("Couldn't fetch Functions due to: ", err))
		}

		newCFs := make(cfunction.CronFunctions, 0)

		for _, function := range functions {
			cF, err := cfunction.ToCronFunction(&function, topic)

			if err != nil {
				log.Print(err)
				continue
			}

			newCFs = append(newCFs, cF)

			// Schedule new entries
			if _, ok := jobEntries[cF.Name]; !ok {
				eID, err := cronScheduler.AddCronFunction(&cF, invoker)
				log.Print("added new function ", cF.Name)

				if err != nil {
					return err
				}

				jobEntries[cF.Name] = eID

				// Update schedule of entries
			}
		}

		log.Print("Functions are ", newCFs)

		// Delete old entries
		for _, f := range cFs {
			if !newCFs.Contains(&f) {
				cronScheduler.Remove(jobEntries[f.Name])
				log.Print("deleted function ", f.Name)
				delete(jobEntries, f.Name)
			}
		}

		cFs = newCFs
	}
}
