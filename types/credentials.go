// Copyright (c) OpenFaaS Author(s) 2024. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package types

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"os"
	"strings"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/openfaas/faas-provider/auth"
	sdk "github.com/openfaas/go-sdk"
)

// GetClientAuth returns authentication credentials for OpenFaaS. The appropriate credentials are returned based on
// the configured authentication mode. If basic_auth=true basic auth credentials are returned. If system_issuer is configured,
// access token credentials are returned. Empty credentials are returned of non of the previous modes is configured.
// An error is returned if obtaining the credentials fails.
func GetClientAuth() (sdk.ClientAuth, error) {

	if val, ok := os.LookupEnv("basic_auth"); ok && len(val) > 0 {
		if val == "true" || val == "1" {
			return getBasicAuthCredentials()
		}
	}

	return nil, nil
}

func getBasicAuthCredentials() (sdk.ClientAuth, error) {
	if _, ok := os.LookupEnv("secret_mount_path"); ok {
		reader := auth.ReadBasicAuthFromDisk{}
		reader.SecretMountPath = os.Getenv("secret_mount_path")

		creds, err := reader.Read()
		if err != nil {
			return nil, err
		}

		return &sdk.BasicAuth{
			Username: creds.User,
			Password: creds.Password,
		}, nil
	}

	if _, err := os.Stat("/var/run/secrets/kubernetes.io"); err != nil && os.IsNotExist(err) {
		username := "admin"
		creds := &auth.BasicAuthCredentials{
			User:     username,
			Password: LookupPasswordViaKubectl(),
		}

		return &sdk.BasicAuth{
			Username: creds.User,
			Password: creds.Password,
		}, nil
	}

	return nil, fmt.Errorf("no basic auth credentials provided")
}

func LookupPasswordViaKubectl() string {

	ctx := context.Background()
	cmd := execute.ExecTask{
		Command:      "kubectl",
		Args:         []string{"get", "secret", "-n", "openfaas", "basic-auth", "-o", "jsonpath='{.data.basic-auth-password}'"},
		StreamStdio:  false,
		PrintCommand: false,
	}

	res, err := cmd.Execute(ctx)
	if err != nil {
		panic(err)
	}

	if res.ExitCode != 0 {
		panic("Non-zero exit code: " + res.Stderr)
	}
	resOut := strings.Trim(res.Stdout, "\\'")

	decoded, err := b64.StdEncoding.DecodeString(resOut)
	if err != nil {
		panic(err)
	}
	password := strings.TrimSpace(string(decoded))

	return password
}
