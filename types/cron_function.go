// Copyright (c) OpenFaaS Author(s) 2020. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package types

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/openfaas/connector-sdk/types"
	ptypes "github.com/openfaas/faas-provider/types"
	"github.com/pkg/errors"
)

// CronFunction depicts an OpenFaaS function which is invoked by cron
type CronFunction struct {
	FuncData  ptypes.FunctionStatus
	Name      string
	Namespace string
	Schedule  string
	Async     bool
}

// CronFunctions a list of CronFunction
type CronFunctions []CronFunction

// Contains returns true if the provided CronFunction object is in list
func (c *CronFunctions) Contains(cF *CronFunction) bool {

	for _, f := range *c {

		if f.Name == cF.Name && f.Namespace == cF.Namespace && f.Schedule == cF.Schedule && f.Async == cF.Async {
			return true
		}

	}

	return false
}

// ToCronFunction converts a ptypes.FunctionStatus object to the CronFunction and returns error if it is not possible
func ToCronFunction(f ptypes.FunctionStatus, namespace string, topic string) (CronFunction, error) {
	if f.Annotations == nil {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has no annotations."))
	}

	fTopic := (*f.Annotations)["topic"]
	fSchedule := (*f.Annotations)["schedule"]
	fAsync := (*f.Annotations)["async"]

	if fAsync != "true" {
		fAsync = "false"
	}

	if fTopic != topic {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has wrong topic: ", fTopic))
	}

	if !CheckSchedule(fSchedule) {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has wrong cron schedule: ", fSchedule))
	}

	if fAsync != "true" && fAsync != "false" {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has invalid async value: ", fAsync))
	}

	var c CronFunction
	c.FuncData = f
	c.Name = f.Name
	c.Namespace = namespace
	c.Schedule = fSchedule
	c.Async, _ = strconv.ParseBool(fAsync)
	return c, nil
}

// InvokeFunction Invokes the cron function
func (c CronFunction) InvokeFunction(i *types.Invoker) (*[]byte, error) {
	gwURL := fmt.Sprintf("%s/function/%s.%s", i.GatewayURL, c.Name, c.Namespace)
	if c.Async {
		gwURL = fmt.Sprintf("%s/async-function/%s.%s", i.GatewayURL, c.Name, c.Namespace)
	}
	reader := bytes.NewReader(make([]byte, 0))
	httpReq, _ := http.NewRequest(http.MethodPost, gwURL, reader)

	if httpReq.Body != nil {
		defer httpReq.Body.Close()
	}

	var body *[]byte
	res, doErr := i.Client.Do(httpReq)

	if doErr != nil {
		i.Responses <- types.InvokerResponse{
			Error: errors.Wrap(doErr, fmt.Sprint("unable to invoke ", c.Name, " in ", c.Namespace)),
		}
		return nil, doErr
	}

	if res.Body != nil {
		defer res.Body.Close()
		bytesOut, readErr := ioutil.ReadAll(res.Body)

		if readErr != nil {
			log.Printf("Error reading body")
			i.Responses <- types.InvokerResponse{
				Error: errors.Wrap(readErr, fmt.Sprint("unable to invoke ", c.Name, " in ", c.Namespace)),
			}
			return nil, doErr
		}

		body = &bytesOut
	}

	i.Responses <- types.InvokerResponse{
		Body:     body,
		Status:   res.StatusCode,
		Header:   &res.Header,
		Function: c.Name,
		Topic:    (*c.FuncData.Annotations)["topic"],
	}

	return body, nil
}

// CronFunctionInterface defines an interface to work with CronFunction during testing
type CronFunctionInterface interface {
	InvokeFunction(i *types.Invoker) (*[]byte, error)
}
