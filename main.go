// Copyright (c) Rishabh Gupta 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/openfaas-incubator/connector-sdk/types"
	"github.com/openfaas/faas/gateway/requests"
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
	runningFuncs := make(cfunction.ScheduledFunctions, 0)
	lookupBuilder := cfunction.FunctionLookupBuilder{
		GatewayURL:  c.Config.GatewayURL,
		Client:      types.MakeClient(c.Config.UpstreamTimeout),
		Credentials: c.Credentials,
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		functions, err := lookupBuilder.GetFunctions()

		if err != nil {
			return errors.New(fmt.Sprint("Couldn't fetch Functions due to: ", err))
		}

		newCronFunctions := RequestsToCronFunctions(functions, topic)
		addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, runningFuncs)

		for _, function := range deleteFuncs {
			cronScheduler.Remove(function)
			log.Print("deleted function ", function.Function.Name)
		}

		newScheduledFuncs := make(cfunction.ScheduledFunctions, 0)

		for _, function := range addFuncs {
			f, err := cronScheduler.AddCronFunction(&function, invoker)
			if err != nil {
				log.Fatal("could not add function ", function.Name)
			}

			newScheduledFuncs = append(newScheduledFuncs, f)
			log.Print("added function ", function.Name)
		}

		runningFuncs = UpdateScheduledFunctions(runningFuncs, newScheduledFuncs, deleteFuncs)
	}
}

// RequestsToCronFunctions converts an array of requests.Function object to CronFunction, ignoring those that cannot be converted
func RequestsToCronFunctions(functions []requests.Function, topic string) cfunction.CronFunctions {
	newCronFuncs := make(cfunction.CronFunctions, 0)
	for _, function := range functions {
		cF, err := cfunction.ToCronFunction(function, topic)
		if err != nil {
			continue
		}
		newCronFuncs = append(newCronFuncs, cF)
	}
	return newCronFuncs
}

// GetNewAndDeleteFuncs takes new functions and running cron functions and returns functions that need to be added and that need to be deleted
func GetNewAndDeleteFuncs(newFuncs cfunction.CronFunctions, oldFuncs cfunction.ScheduledFunctions) (cfunction.CronFunctions, cfunction.ScheduledFunctions) {
	addFuncs := make(cfunction.CronFunctions, 0)
	deleteFuncs := make(cfunction.ScheduledFunctions, 0)

	for _, function := range newFuncs {
		if !oldFuncs.Contains(&function) {
			addFuncs = append(addFuncs, function)
		}
	}

	for _, function := range oldFuncs {
		if !newFuncs.Contains(&function.Function) {
			deleteFuncs = append(deleteFuncs, function)
		}
	}

	return addFuncs, deleteFuncs
}

// UpdateScheduledFunctions updates the scheduled function with added functions and removes deleted functions
func UpdateScheduledFunctions(running, added, deleted cfunction.ScheduledFunctions) cfunction.ScheduledFunctions {
	updatedSchedule := make(cfunction.ScheduledFunctions, 0)

	for _, function := range running {
		if !deleted.Contains(&function.Function) {
			updatedSchedule = append(updatedSchedule, function)
		}
	}

	for _, function := range added {
		updatedSchedule = append(updatedSchedule, function)
	}

	return updatedSchedule
}
