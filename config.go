package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/openfaas/connector-sdk/types"
)

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

	var printResponseBody bool
	if val, exists := os.LookupEnv("print_response_body"); exists {
		printResponseBody = (val == "1" || val == "true")
	}

	rebuildInterval := time.Second * 10

	if val, exists := os.LookupEnv("rebuild_interval"); exists {
		d, err := time.ParseDuration(val)
		if err != nil {
			return nil, err
		}
		rebuildInterval = d
	}

	var basicAuth bool
	if val, exists := os.LookupEnv("basic_auth"); exists {
		a, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}

		basicAuth = a
	}

	return &types.ControllerConfig{
		RebuildInterval:         rebuildInterval,
		GatewayURL:              gURL,
		AsyncFunctionInvocation: asynchronousInvocation,
		ContentType:             contentType,
		PrintResponse:           true,
		PrintResponseBody:       printResponseBody,
		PrintRequestBody:        false,
		BasicAuth:               basicAuth,
	}, nil
}
