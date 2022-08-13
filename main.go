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
	crontypes "github.com/openfaas/cron-connector/types"
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
		config.PrintRequestBody)

	go func() {
		for {
			r := <-invoker.Responses
			if r.Error != nil {
				log.Printf("Error with: %s, %s", r.Function, err.Error())
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

	cronScheduler := crontypes.NewScheduler()
	cronScheduler.Start()
	if err := startFunctionProbe(config.RebuildInterval, rebuildTimeout, topic, config, cronScheduler, invoker); err != nil {
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

func startFunctionProbe(interval time.Duration, probeTimeout time.Duration, topic string, c *types.ControllerConfig, cronScheduler *crontypes.Scheduler, invoker *types.Invoker) error {
	runningFuncs := make(crontypes.ScheduledFunctions, 0)

	creds := types.GetCredentials()
	auth := &BasicAuth{
		Username: creds.User,
		Password: creds.Password,
	}

	sdkClient, err := sdk.NewClient(auth, c.GatewayURL, nil, &probeTimeout)
	if err != nil {
		return err
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
				log.Printf("Removed: %s [%s]",
					function.Function.String(),
					function.Function.Schedule)

				cronScheduler.Remove(function)
			}

			newScheduledFuncs := make(crontypes.ScheduledFunctions, 0)

			for _, function := range addFuncs {
				f, err := cronScheduler.AddCronFunction(function, invoker)
				if err != nil {
					return fmt.Errorf("can't add function: %s, %w", function.String(), err)
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
