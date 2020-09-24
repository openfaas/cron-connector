// Copyright (c) OpenFaaS Author(s) 2020. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
package main

import (
	"testing"

	cfunction "github.com/openfaas/cron-connector/types"
	ptypes "github.com/openfaas/faas-provider/types"
)

func TestGetNewAndDeleteFuncs(t *testing.T) {
	newCronFunctions := make(cfunction.CronFunctions, 3)
	defaultReq := ptypes.FunctionStatus{}
	newCronFunctions[0] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_unchanged", Namespace: "openfaas-fn", Schedule: "* * * * *"}
	newCronFunctions[1] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_add", Namespace: "openfaas-fn", Schedule: "* * * * *"}
	newCronFunctions[2] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_update", Namespace: "openfaas-fn", Schedule: "*/5 * * * *"}

	oldFuncs := make(cfunction.ScheduledFunctions, 3)
	oldFuncs[0] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_unchanged", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[1] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_delete", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[2] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_update", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}

	addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, oldFuncs, "openfaas-fn")
	if !deleteFuncs.Contains(&oldFuncs[1].Function) {
		t.Error("function was not deleted")
	}

	if !addFuncs.Contains(&newCronFunctions[1]) {
		t.Error("function was not added")
	}

	if !deleteFuncs.Contains(&oldFuncs[2].Function) && !addFuncs.Contains(&newCronFunctions[2]) {
		t.Error("function will not be updated")
	}

	if addFuncs.Contains(&newCronFunctions[0]) || deleteFuncs.Contains(&newCronFunctions[0]) {
		t.Error("function should be left as it is")
	}
}

func TestNamespaceFuncs(t *testing.T) {
	newCronFunctions := make(cfunction.CronFunctions, 3)
	defaultReq := ptypes.FunctionStatus{}
	newCronFunctions[0] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_one", Namespace: "openfaas-fn", Schedule: "* * * * *"}
	newCronFunctions[1] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_one", Namespace: "custom", Schedule: "* * * * *"}
	newCronFunctions[2] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_update", Namespace: "openfaas-fn", Schedule: "*/5 * * * *"}

	oldFuncs := make(cfunction.ScheduledFunctions, 3)
	oldFuncs[0] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_one", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[1] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_delete", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[2] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_function_to_update", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}

	addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, oldFuncs, "openfaas-fn")
	if !deleteFuncs.Contains(&oldFuncs[1].Function) {
		t.Error("function was not deleted")
	}

	if !addFuncs.Contains(&newCronFunctions[1]) {
		t.Error("function was not added")
	}

	if !deleteFuncs.Contains(&oldFuncs[2].Function) && !addFuncs.Contains(&newCronFunctions[2]) {
		t.Error("function will not be updated")
	}

	if addFuncs.Contains(&newCronFunctions[0]) || deleteFuncs.Contains(&newCronFunctions[0]) {
		t.Error("function should be left as it is")
	}
}

func TestAsyncFuncs(t *testing.T) {
	newCronFunctions := make(cfunction.CronFunctions, 3)
	defaultReq := ptypes.FunctionStatus{}
	newCronFunctions[0] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_one", Namespace: "openfaas-fn", Schedule: "* * * * *"}
	newCronFunctions[1] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_two", Namespace: "openfaas-fn", Schedule: "* * * * *", Async: true}
	newCronFunctions[2] = cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_three", Namespace: "openfaas-fn", Schedule: "* * * * *", Async: true}

	oldFuncs := make(cfunction.ScheduledFunctions, 3)
	oldFuncs[0] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_one", Namespace: "openfaas-fn", Schedule: "* * * * *"}, ID: 0}
	oldFuncs[1] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_two", Namespace: "openfaas-fn", Schedule: "* * * * *", Async: false}, ID: 0}
	oldFuncs[1] = cfunction.ScheduledFunction{Function: cfunction.CronFunction{FuncData: defaultReq, Name: "test_async_three", Namespace: "openfaas-fn", Schedule: "* * * * *", Async: true}, ID: 0}

	addFuncs, deleteFuncs := GetNewAndDeleteFuncs(newCronFunctions, oldFuncs, "openfaas-fn")
	if !deleteFuncs.Contains(&oldFuncs[1].Function) && !addFuncs.Contains(&newCronFunctions[1]) {
		t.Error("function will not be updated")
	}

	if addFuncs.Contains(&newCronFunctions[0]) || deleteFuncs.Contains(&newCronFunctions[0]) {
		t.Error("function should be left as it is")
	}

	if addFuncs.Contains(&newCronFunctions[2]) || deleteFuncs.Contains(&newCronFunctions[2]) {
		t.Error("function should be left as it is")
	}
}
