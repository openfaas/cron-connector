// Copyright (c) OpenFaaS Author(s) 2021. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	sdk "github.com/openfaas/go-sdk"

	"github.com/openfaas/connector-sdk/types"
	crontypes "github.com/openfaas/cron-connector/types"
	"github.com/openfaas/cron-connector/version"
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

	rebuildTimeout := time.Second * 5
	if val, exists := os.LookupEnv("rebuild_timeout"); exists {
		rebuildTimeout, err = time.ParseDuration(val)
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
	}

	sha, ver := version.GetReleaseInfo()
	log.Printf("Version: %s\tCommit: %s\n", sha, ver)
	log.Printf("Gateway URL: %s", config.GatewayURL)
	log.Printf("Async Invocation: %v", config.AsyncFunctionInvocation)
	log.Printf("Rebuild interval: %s\tRebuild timeout: %s", config.RebuildInterval, rebuildTimeout)

	httpClient := types.MakeClient(config.UpstreamTimeout)
	invoker := types.NewInvoker(
		gatewayRoute(config),
		httpClient,
		config.ContentType,
		config.PrintResponse,
		config.PrintRequestBody,
		"openfaas-ce/cron-connector")

	go func() {
		for {
			r := <-invoker.Responses
			if r.Error != nil {
				log.Printf("Error with %s: %s", r.Function, r.Error)
			} else {
				duration := fmt.Sprintf("%.2fs", r.Duration.Seconds())
				if r.Duration < time.Second*1 {
					duration = fmt.Sprintf("%dms", r.Duration.Milliseconds())
				}
				log.Printf("Response: %s [%d] (%s)",
					r.Function,
					r.Status,
					duration)
			}
		}
	}()

	auth, err := crontypes.GetClientAuth()
	if err != nil {
		log.Fatalf("Failed to get auth credentials: %s", err)
	}

	cronScheduler := crontypes.NewScheduler()
	cronScheduler.Start()

	u, err := url.Parse(config.GatewayURL)
	if err != nil {
		log.Fatalf("Failed to parse gateway URL: %s", err)
	}

	if err := startFunctionProbe(u, config.RebuildInterval, rebuildTimeout, topic, config, cronScheduler, invoker, auth); err != nil {
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

func startFunctionProbe(gatewayURL *url.URL, interval time.Duration, probeTimeout time.Duration, topic string, c *types.ControllerConfig, cronScheduler *crontypes.Scheduler, invoker *types.Invoker, auth sdk.ClientAuth) error {
	runningFuncs := make(crontypes.ScheduledFunctions, 0)

	httpClient := &http.Client{}
	httpClient.Timeout = probeTimeout

	sdkClient := sdk.NewClient(gatewayURL, auth, httpClient)

	ctx := context.Background()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		namespaces, err := sdkClient.GetNamespaces(ctx)
		if err != nil {
			log.Printf("error listing namespaces: %s", err)
			continue
		}

		for _, namespace := range namespaces {
			functions, err := sdkClient.GetFunctions(ctx, namespace)
			if err != nil {
				log.Printf("error listing functions in %s: %s", namespace, err)
				continue
			}

			newCronFunctions := requestsToCronFunctions(functions, namespace, topic)
			addFuncs, deleteFuncs := getNewAndDeleteFuncs(newCronFunctions, runningFuncs, namespace)

			for _, function := range deleteFuncs {
				log.Printf("Removed: %s [%s]",
					function.Function.String(),
					function.Function.Schedule)

				cronScheduler.Remove(function)
			}

			newScheduledFuncs := make(crontypes.ScheduledFunctions, 0)

			for _, function := range addFuncs {
				f, err := cronScheduler.AddCronFunction(function, invoker)
				if err != nil {
					log.Printf("can't add function: %s, %s", function.String(), err)
					continue
				}

				newScheduledFuncs = append(newScheduledFuncs, f)
				log.Printf("Added: %s [%s]", function.String(), function.Schedule)
			}

			runningFuncs = updateScheduledFunctions(runningFuncs, newScheduledFuncs, deleteFuncs)
		}
	}
}

// requestsToCronFunctions converts an array of types.FunctionStatus object
// to CronFunction, ignoring those that cannot be converted
func requestsToCronFunctions(functions []ptypes.FunctionStatus, namespace string, topic string) crontypes.CronFunctions {
	newCronFuncs := make(crontypes.CronFunctions, 0)
	for _, function := range functions {
		cF, err := crontypes.ToCronFunction(function, namespace, topic)
		if err != nil {
			continue
		}
		newCronFuncs = append(newCronFuncs, cF)
	}
	return newCronFuncs
}

// getNewAndDeleteFuncs takes new functions and running cron functions and returns
// functions that need to be added and that need to be deleted
func getNewAndDeleteFuncs(newFuncs crontypes.CronFunctions, oldFuncs crontypes.ScheduledFunctions, namespace string) (crontypes.CronFunctions, crontypes.ScheduledFunctions) {
	addFuncs := make(crontypes.CronFunctions, 0)
	deleteFuncs := make(crontypes.ScheduledFunctions, 0)

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
func updateScheduledFunctions(running, added, deleted crontypes.ScheduledFunctions) crontypes.ScheduledFunctions {
	updatedSchedule := make(crontypes.ScheduledFunctions, 0)

	for _, function := range running {
		if !deleted.Contains(&function.Function) {
			updatedSchedule = append(updatedSchedule, function)
		}
	}

	updatedSchedule = append(updatedSchedule, added...)

	return updatedSchedule
}
