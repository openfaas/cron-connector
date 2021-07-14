// Copyright (c) OpenFaaS Author(s) 2021. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openfaas/connector-sdk/types"
	cfunction "github.com/openfaas/cron-connector/types"
	"github.com/openfaas/cron-connector/version"
	sdk "github.com/openfaas/faas-cli/proxy"
	ptypes "github.com/openfaas/faas-provider/types"
)

// topic is the value of the "topic" annotation to look for
// on functions, to decide to include them for invocation
const topic = "cron-function"

func main() {
	config, err := getControllerConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	sha, ver := version.GetReleaseInfo()
	log.Printf("Version: %s\tCommit: %s\n", sha, ver)
	log.Printf("Gateway URL: %s", config.GatewayURL)
	log.Printf("Async Invocation: %v", config.AsyncFunctionInvocation)

	invoker := types.NewInvoker(gatewayRoute(config),
		types.MakeClient(config.UpstreamTimeout),
		config.ContentType,
		config.PrintResponse)

	go func() {
		for {
			r := <-invoker.Responses
			if r.Error != nil {
				log.Printf("Error with: %s, %s", r.Function, err.Error())
			} else {
				log.Printf("Response: %s [%d]", r.Function, r.Status)
			}
		}
	}()

	cronScheduler := cfunction.NewScheduler()
	interval := time.Second * 10

	cronScheduler.Start()
	err = startFunctionProbe(interval, topic, config, cronScheduler, invoker)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func gatewayRoute(config *types.ControllerConfig) string {
	if config.AsyncFunctionInvocation {
		return fmt.Sprintf("%s/%s", config.GatewayURL, "async-function")
	}

	return fmt.Sprintf("%s/%s", config.GatewayURL, "function")
}

func getControllerConfig() (*types.ControllerConfig, error) {
	gURL, ok := os.LookupEnv("gateway_url")

	if !ok {
		return nil, fmt.Errorf("gateway_url environment variable not set")
	}

	asynchronousInvocation := false
	if val, exists := os.LookupEnv("asynchronous_invocation"); exists {
		asynchronousInvocation = (val == "1" || val == "true")
	}

	contentType := "text/plain"
	if v, exists := os.LookupEnv("content_type"); exists && len(v) > 0 {
		contentType = v
	}

	return &types.ControllerConfig{
		RebuildInterval:         time.Millisecond * 1000,
		GatewayURL:              gURL,
		PrintResponse:           true,
		AsyncFunctionInvocation: asynchronousInvocation,
		ContentType:             contentType,
	}, nil
}

// BasicAuth basic authentication for the the gateway
type BasicAuth struct {
	Username string
	Password string
}

// Set set Authorization header on request
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
			return fmt.Errorf("can't list namespaces: %w", err)
		}

		for _, namespace := range namespaces {
			functions, err := sdkClient.ListFunctions(ctx, namespace)
			if err != nil {
				return fmt.Errorf("can't list functions: %w", err)
			}

			newCronFunctions := requestsToCronFunctions(functions, namespace, topic)
			addFuncs, deleteFuncs := getNewAndDeleteFuncs(newCronFunctions, runningFuncs, namespace)

			for _, function := range deleteFuncs {
				log.Printf("Unregistered [%s]", function.Function.String())

				cronScheduler.Remove(function)
			}

			newScheduledFuncs := make(cfunction.ScheduledFunctions, 0)

			for _, function := range addFuncs {
				f, err := cronScheduler.AddCronFunction(function, invoker)
				if err != nil {
					return fmt.Errorf("can't add function: %s, %w", function.String(), err)
				}

				newScheduledFuncs = append(newScheduledFuncs, f)
				log.Printf("Registered: %s [%s]", function.String(), function.Schedule)
			}

			runningFuncs = updateScheduledFunctions(runningFuncs, newScheduledFuncs, deleteFuncs)
		}
	}
}

// requestsToCronFunctions converts an array of types.FunctionStatus object
// to CronFunction, ignoring those that cannot be converted
func requestsToCronFunctions(functions []ptypes.FunctionStatus, namespace string, topic string) cfunction.CronFunctions {
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

// getNewAndDeleteFuncs takes new functions and running cron functions and returns
// functions that need to be added and that need to be deleted
func getNewAndDeleteFuncs(newFuncs cfunction.CronFunctions, oldFuncs cfunction.ScheduledFunctions, namespace string) (cfunction.CronFunctions, cfunction.ScheduledFunctions) {
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

// updateScheduledFunctions updates the scheduled function with
// added functions and removes deleted functions
func updateScheduledFunctions(running, added, deleted cfunction.ScheduledFunctions) cfunction.ScheduledFunctions {
	updatedSchedule := make(cfunction.ScheduledFunctions, 0)

	for _, function := range running {
		if !deleted.Contains(&function.Function) {
			updatedSchedule = append(updatedSchedule, function)
		}
	}

	updatedSchedule = append(updatedSchedule, added...)

	return updatedSchedule
}
