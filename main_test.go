// Copyright (c) Rishabh Gupta 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
package main

import (
	"testing"

	cfunction "github.com/zeerorg/cron-connector/types"
)

func TestGetNewAndDeleteFuncs(t *testing.T) {
	newCronFunctions := make(cfunction.CronFunctions, 3)
	newCronFunctions[0] = cfunction.CronFunction{FuncData: nil, Name: "test_function_unchanged", Schedule: "* * * * *"}
	newCronFunctions[1] = cfunction.CronFunction{FuncData: nil, Name: "test_function_to_add", Schedule: "* * * * *"}
	newCronFunctions[1] = cfunction.CronFunction{FuncData: nil, Name: "test_function_to_update", Schedule: "*/5 * * * *"}

	oldFuncs := make(cfunction.ScheduledFunctions, 3)
	oldFuncs[0] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: nil, Name: "test_function_unchanged", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[1] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: nil, Name: "test_function_to_delete", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[2] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: nil, Name: "test_function_to_update", Schedule: "* * * * *"}, ID: 0}

	addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, oldFuncs)
	if !deleteFuncs.Contains(&oldFuncs[1].Function) {
		t.Error("function was not deleted")
	}

	if !addFuncs.Contains(&newCronFunctions[1]) {
		t.Error("function was not added")
	}

	if !deleteFuncs.Contains(&oldFuncs[2].Function) && !addFuncs.Contains(&newCronFunctions[1]) {
		t.Error("function will not be updated")
	}

	if addFuncs.Contains(&newCronFunctions[0]) || deleteFuncs.Contains(&newCronFunctions[0]) {
		t.Error("function should be left as it is")
	}
}
