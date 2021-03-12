// Copyright (c) OpenFaaS Author(s) 2020. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/openfaas/connector-sdk/types"
	cfunction "github.com/openfaas/cron-connector/types"
	"github.com/openfaas/cron-connector/version"
	sdk "github.com/openfaas/faas-cli/proxy"
	ptypes "github.com/openfaas/faas-provider/types"
)

func main() {
	config, err := getControllerConfig()
	if err != nil {
		panic(err)
	}

	sha, ver := version.GetReleaseInfo()
	log.Printf("Version: %s\tCommit: %s", sha, ver)

	context := "/async-function"
	if !config.AsyncFunctionInvocation {
		context = "/function"
	}

	invoker := types.NewInvoker(config.GatewayURL+context, types.MakeClient(config.UpstreamTimeout), config.PrintResponse)
	cronScheduler := cfunction.NewScheduler()
	topic := "cron-function"
	interval := time.Second * 10

	cronScheduler.Start()
	err = startFunctionProbe(interval, topic, config, cronScheduler, invoker)

	if err != nil {
		panic(err)
	}
}

func getControllerConfig() (*types.ControllerConfig, error) {
	gURL, ok := os.LookupEnv("gateway_url")

	if !ok {
		return nil, fmt.Errorf("Gateway URL not set")
	}

	// Get the async env value, if present. Defaults to true.
	async, ok := os.LookupEnv("async_invocation")
	if !ok {
		async = "true"
	}
	asyncParsed, err := strconv.ParseBool(async)
	if err != nil {
		asyncParsed = true
	}

	return &types.ControllerConfig{
		GatewayURL:              gURL,
		RebuildInterval:         time.Millisecond * 1000,
		AsyncFunctionInvocation: asyncParsed,
		PrintResponse:           true,
	}, nil
}

//BasicAuth basic authentication for the the gateway
type BasicAuth struct {
	Username string
	Password string
}

//Set set Authorization header on request
func (auth *BasicAuth) Set(req *http.Request) error {
	req.SetBasicAuth(auth.Username, auth.Password)
	return nil
}

func startFunctionProbe(interval time.Duration, topic string, c *types.ControllerConfig, cronScheduler *cfunction.Scheduler, invoker *types.Invoker) error {
	runningFuncs := make(cfunction.ScheduledFunctions, 0)
	timeout := 3 * time.Second
	auth := &BasicAuth{}
	auth.Username = types.GetCredentials().User
	auth.Password = types.GetCredentials().Password

	sdkClient, err := sdk.NewClient(auth, c.GatewayURL, nil, &timeout)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		namespaces, err := sdkClient.ListNamespaces(ctx)
		if err != nil {
			return fmt.Errorf("Couldn't fetch Namespaces due to: %s", err)
		}

		for _, namespace := range namespaces {
			functions, err := sdkClient.ListFunctions(ctx, namespace)
			if err != nil {
				return fmt.Errorf("Couldn't fetch Functions due to: %s", err)
			}

			newCronFunctions := RequestsToCronFunctions(functions, namespace, topic)
			addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, runningFuncs, namespace)

			for _, function := range deleteFuncs {
				cronScheduler.Remove(function)
				log.Print("deleted function ", function.Function.Name, " in ", function.Function.Namespace)
			}

			newScheduledFuncs := make(cfunction.ScheduledFunctions, 0)

			for _, function := range addFuncs {
				f, err := cronScheduler.AddCronFunction(function, invoker)
				if err != nil {
					log.Fatal("could not add function ", function.Name, " in ", function.Namespace)
				}

				newScheduledFuncs = append(newScheduledFuncs, f)
				log.Print("added function ", function.Name, " in ", function.Namespace)
			}

			runningFuncs = UpdateScheduledFunctions(runningFuncs, newScheduledFuncs, deleteFuncs)
		}
	}
}

// RequestsToCronFunctions converts an array of types.FunctionStatus object to CronFunction, ignoring those that cannot be converted
func RequestsToCronFunctions(functions []ptypes.FunctionStatus, namespace string, topic string) cfunction.CronFunctions {
	newCronFuncs := make(cfunction.CronFunctions, 0)
	for _, function := range functions {
		cF, err := cfunction.ToCronFunction(function, namespace, topic)
		if err != nil {
			continue
		}
		newCronFuncs = append(newCronFuncs, cF)
	}
	return newCronFuncs
}

// GetNewAndDeleteFuncs takes new functions and running cron functions and returns functions that need to be added and that need to be deleted
func GetNewAndDeleteFuncs(newFuncs cfunction.CronFunctions, oldFuncs cfunction.ScheduledFunctions, namespace string) (cfunction.CronFunctions, cfunction.ScheduledFunctions) {
	addFuncs := make(cfunction.CronFunctions, 0)
	deleteFuncs := make(cfunction.ScheduledFunctions, 0)

	for _, function := range newFuncs {
		if !oldFuncs.Contains(&function) {
			addFuncs = append(addFuncs, function)
		}
	}

	for _, function := range oldFuncs {
		if !newFuncs.Contains(&function.Function) && function.Function.Namespace == namespace {
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
