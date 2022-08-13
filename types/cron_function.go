// Copyright (c) OpenFaaS Author(s) 2021. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package types

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/openfaas/connector-sdk/types"
	ptypes "github.com/openfaas/faas-provider/types"
)

// CronFunction depicts an OpenFaaS function which is invoked by cron
type CronFunction struct {
	FuncData  ptypes.FunctionStatus
	Name      string
	Namespace string
	Schedule  string
}

func (c *CronFunction) String() string {
	if len(c.Namespace) > 0 {
		return fmt.Sprintf("%s.%s", c.Name, c.Namespace)
	}

	return c.Name
}

// CronFunctions a list of CronFunction
type CronFunctions []CronFunction

// Contains returns true if the provided CronFunction object is in list
func (c *CronFunctions) Contains(cf *CronFunction) bool {
	for _, f := range *c {
		if f.Name == cf.Name &&
			f.Namespace == cf.Namespace &&
			f.Schedule == cf.Schedule {
			return true
		}
	}
	return false
}

// ToCronFunction converts a ptypes.FunctionStatus object to the CronFunction
// and returns error if it is not possible
func ToCronFunction(f ptypes.FunctionStatus, namespace string, topic string) (CronFunction, error) {
	if f.Annotations == nil {
		return CronFunction{}, fmt.Errorf("%s has no annotations", f.Name)
	}

	fTopic := (*f.Annotations)["topic"]
	fSchedule := (*f.Annotations)["schedule"]

	if fTopic != topic {
		return CronFunction{}, fmt.Errorf("%s has wrong topic: %s", fTopic, f.Name)
	}

	if !CheckSchedule(fSchedule) {
		return CronFunction{}, fmt.Errorf("%s has wrong cron schedule: %s", f.Name, fSchedule)
	}

	return CronFunction{
		FuncData:  f,
		Name:      f.Name,
		Namespace: namespace,
		Schedule:  fSchedule,
	}, nil
}

// InvokeFunction Invokes the cron function
func (c CronFunction) InvokeFunction(i *types.Invoker) (*[]byte, error) {

	name := c.Name
	topic := (*c.FuncData.Annotations)["topic"]

	headers := http.Header{
		"X-Topic":     {topic},
		"X-Connector": {"cron-connector"},
	}

	gwURL := fmt.Sprintf("%s/%s", i.GatewayURL, c.String())

	req, err := http.NewRequest(http.MethodPost, gwURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to %s %w", gwURL, err)
	}

	for k, v := range headers {
		req.Header[k] = v
	}

	if req.Body != nil {
		defer req.Body.Close()
	}
	start := time.Now()

	var body *[]byte
	res, err := i.Client.Do(req)
	if err != nil {
		i.Responses <- types.InvokerResponse{
			Error:    fmt.Errorf("unable to invoke %s %w", c.String(), err),
			Function: name,
			Topic:    topic,
			Status:   http.StatusServiceUnavailable,
			Duration: time.Since(start),
		}
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
		bytesOut, err := ioutil.ReadAll(res.Body)

		if err != nil {
			log.Printf("Error reading body")
			i.Responses <- types.InvokerResponse{
				Error:    fmt.Errorf("unable to invoke %s %w", c.String(), err),
				Status:   http.StatusServiceUnavailable,
				Function: name,
				Topic:    topic,
				Duration: time.Since(start),
			}

			return nil, fmt.Errorf("unable to read body %s", err)
		}

		body = &bytesOut
	}

	i.Responses <- types.InvokerResponse{
		Body:     body,
		Status:   res.StatusCode,
		Header:   &res.Header,
		Function: name,
		Topic:    topic,
		Duration: time.Since(start),
	}

	return body, nil
}
