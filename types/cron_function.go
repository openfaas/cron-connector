// Copyright (c) OpenFaaS Author(s) 2021. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package types

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has no annotations."))
	}
	fTopic := (*f.Annotations)["topic"]
	fSchedule := (*f.Annotations)["schedule"]

	if fTopic != topic {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has wrong topic: ", fTopic))
	}

	if !CheckSchedule(fSchedule) {
		return CronFunction{}, errors.New(fmt.Sprint(f.Name, " has wrong cron schedule: ", fSchedule))
	}

	var c CronFunction
	c.FuncData = f
	c.Name = f.Name
	c.Namespace = namespace
	c.Schedule = fSchedule
	return c, nil
}

// InvokeFunction Invokes the cron function
func (c CronFunction) InvokeFunction(i *types.Invoker) (*[]byte, error) {

	gwURL := fmt.Sprintf("%s/%s", i.GatewayURL, c.String())
	log.Printf("HTTP POST: %s", gwURL)

	req, err := http.NewRequest(http.MethodPost, gwURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	if req.Body != nil {
		defer req.Body.Close()
	}

	var body *[]byte
	res, err := i.Client.Do(req)

	if err != nil {
		i.Responses <- types.InvokerResponse{
			Error: errors.Wrap(err, fmt.Sprint("unable to invoke ", c.Name, " in ", c.Namespace)),
		}
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
		bytesOut, err := ioutil.ReadAll(res.Body)

		if err != nil {
			log.Printf("Error reading body")
			i.Responses <- types.InvokerResponse{
				Error: errors.Wrap(err, fmt.Sprint("unable to invoke ", c.Name, " in ", c.Namespace)),
			}
			return nil, err
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
