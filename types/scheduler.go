// Copyright (c) OpenFaaS Author(s) 2021. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package types

import (
	"log"

	"github.com/openfaas/connector-sdk/types"
	cron "github.com/robfig/cron/v3"
)

// EntryID type redifined for this package
type EntryID cron.EntryID

var standardParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// Scheduler is an interface which talks with cron scheduler
type Scheduler struct {
	main *cron.Cron
}

// ScheduledFunction is a CronFunction that has been scheduled to run
type ScheduledFunction struct {

	// Function is CronFunction object which is running
	Function CronFunction

	// Id is the entryid for the scheduled function
	ID EntryID
}

// ScheduledFunctions is an array of ScheduledFunction
type ScheduledFunctions []ScheduledFunction

// NewScheduler returns a scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron.New(cron.WithParser(standardParser)),
	}
}

// Start a scheduler in new go routine
func (s *Scheduler) Start() {
	s.main.Start()
}

// AddCronFunction adds a function to cron
func (s *Scheduler) AddCronFunction(c CronFunction, invoker *types.Invoker) (ScheduledFunction, error) {
	eID, err := s.main.AddFunc(c.Schedule, func() {
		log.Printf("Executing function: %s", c.String())
		c.InvokeFunction(invoker)
	})
	return ScheduledFunction{c, EntryID(eID)}, err
}

// Remove removes the function from scheduler
func (s *Scheduler) Remove(function ScheduledFunction) {
	s.main.Remove(cron.EntryID(function.ID))
}

// CheckSchedule returns true if the schedule string is compliant with cron
func CheckSchedule(schedule string) bool {
	_, err := standardParser.Parse(schedule)
	return err == nil
}

// Contains returns true if the ScheduledFunctions array contains the CronFunction
func (functions *ScheduledFunctions) Contains(cronFunc *CronFunction) bool {
	for _, f := range *functions {
		if f.Function.Name == cronFunc.Name &&
			f.Function.Namespace == cronFunc.Namespace &&
			f.Function.Schedule == cronFunc.Schedule {
			return true
		}
	}

	return false
}
