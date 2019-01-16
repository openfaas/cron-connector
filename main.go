// Copyright (c) Alex Ellis 2017. All rights reserved.
// Copyright (c) OpenFaaS Project 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/openfaas-incubator/connector-sdk/types"
	cfunction "github.com/zeerorg/cron-connector/types"
	"gopkg.in/robfig/cron.v2"
)

func main() {
	creds := types.GetCredentials()
	config, err := getControllerConfig()

	if err != nil {
		panic(err)
	}

	controller := types.NewController(creds, config)
	cronScheduler := cron.New()
	topic := "cron-function"
	interval := time.Second * 2

	cronScheduler.Start()
	err = startFunctionProbe(interval, topic, controller, cronScheduler, controller.Invoker)
	if err != nil {
		panic(err)
	}

	// How to do it
	// 1. Fetch all Functions with given topic ("cron-function")
	// 2. Fetch their names and schedule
	// 3. Compare the names and schedule with the existing functions.
	// 4. Remove Deleted functions from scheduler
	// 5. Add new functions to the scheduler
	// 6. Repeat
	// To keep in mind: Don't touch previous jobs

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

func startFunctionProbe(interval time.Duration, topic string, c *types.Controller, cronScheduler *cron.Cron, invoker *types.Invoker) error {
	cFs := make(cfunction.CronFunctions, 0)
	lookupBuilder := cfunction.FunctionLookupBuilder{
		GatewayURL:  c.Config.GatewayURL,
		Client:      types.MakeClient(c.Config.UpstreamTimeout),
		Credentials: c.Credentials,
	}
	ticker := time.NewTicker(interval)
	jobEntries := make(map[string]cron.EntryID)

	defer ticker.Stop()

	for {
		<-ticker.C
		functions, err := lookupBuilder.GetFunctions()

		if err != nil {
			log.Fatal("Couldn't fetch Functions due to: ", err)
			continue
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
				eID, err := cronScheduler.AddFunc(cF.Schedule, func() { cF.InvokeFunction(invoker) })

				if err != nil {
					return err
				}

				jobEntries[cF.Name] = eID
			} else {
				for _, tempF := range cFs {
					if tempF.Name == cF.Name && tempF.Schedule != cF.Schedule {
						cronScheduler.Remove(jobEntries[cF.Name])
						eID, err := cronScheduler.AddFunc(cF.Schedule, func() { cF.InvokeFunction(invoker) })

						if err != nil {
							return err
						}

						jobEntries[cF.Name] = eID
					}
				}
			}
		}

		// Delete old entries
		for _, f := range cFs {
			if !newCFs.Contains(&f) {
				cronScheduler.Remove(jobEntries[f.Name])
				delete(jobEntries, f.Name)
			}
		}
	}
}
