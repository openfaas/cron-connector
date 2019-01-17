package types

import (
	"fmt"
	"log"

	"github.com/openfaas-incubator/connector-sdk/types"
	"gopkg.in/robfig/cron.v2"
)

// EntryID type redifined for this package
type EntryID cron.EntryID

// Scheduler is an interface which talks with cron scheduler
type Scheduler struct {
	main *cron.Cron
}

// AddCronFunction adds a function to cron
func (s *Scheduler) AddCronFunction(c *CronFunction, invoker *types.Invoker) (EntryID, error) {
	eID, err := s.main.AddFunc(c.Schedule, func() {
		log.Print(fmt.Sprint("Executed function: ", c.Name))
		c.InvokeFunction(invoker)
	})
	return EntryID(eID), err
}

// NewScheduler returns a scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron.New(),
	}
}

// Start a scheduler in new go routine
func (s *Scheduler) Start() {
	s.main.Start()
}

// Remove removes the function from scheduler
func (s *Scheduler) Remove(id EntryID) {
	s.main.Remove(cron.EntryID(id))
}

// CheckSchedule returns true if the schedule string is compliant with cron
func CheckSchedule(schedule string) bool {
	_, err := cron.Parse(schedule)
	return err == nil
}
