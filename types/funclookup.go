package types

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/openfaas-incubator/connector-sdk/types"
	"github.com/openfaas/faas/gateway/requests"
)

// FunctionLookupBuilder alias for types.FunctionLookupBuilder
type FunctionLookupBuilder types.FunctionLookupBuilder

// GetFunctions requests the OpenFaas gteway to return a list of all functions
func (lookup *FunctionLookupBuilder) GetFunctions() ([]requests.Function, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/system/functions", lookup.GatewayURL), nil)

	if lookup.Credentials != nil {
		req.SetBasicAuth(lookup.Credentials.User, lookup.Credentials.Password)
	}

	res, reqErr := lookup.Client.Do(req)

	if reqErr != nil {
		return nil, reqErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	bytesOut, _ := ioutil.ReadAll(res.Body)

	functions := []requests.Function{}
	marshalErr := json.Unmarshal(bytesOut, &functions)

	if marshalErr != nil {
		return nil, marshalErr
	}

	return functions, nil
}
