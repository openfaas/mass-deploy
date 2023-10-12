package main

import (
	"context"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/alexellis/go-execute/v2"
	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/go-sdk"
)

func main() {

	var (
		gateway, username     string
		action                string
		name, image, fprocess string
		functions, startAt    int
	)

	flag.StringVar(&gateway, "gateway", "http://127.0.0.1:8080", "gateway url")

	flag.IntVar(&functions, "functions", 100, "number of functions to create")
	flag.IntVar(&startAt, "start-at", 0, "start at function number")
	flag.StringVar(&name, "name", "env", "name of function")
	flag.StringVar(&image, "image", "", "image to use for function")
	flag.StringVar(&fprocess, "fprocess", "env", "fprocess to use for function")
	flag.StringVar(&action, "action", "create", "action to perform")

	flag.Parse()

	gatewayURL, _ := url.Parse(gateway)

	password := lookupPasswordViaKubectl()

	username = "admin"

	auth := &sdk.BasicAuth{
		Username: username,
		Password: password,
	}

	if len(image) == 0 {
		panic("-image is required")
	}

	client := sdk.NewClient(gatewayURL, auth, http.DefaultClient)

	for i := startAt; i < functions; i++ {
		name := fmt.Sprintf("%s-%d", name, i+1)

		if action == "create" {
			log.Printf("Creating function %d: %s", i+1, name)
			spec := types.FunctionDeployment{
				Service: name,
				Image:   image,
			}

			if len(fprocess) > 0 {
				spec.EnvProcess = fprocess
			}

			code, err := client.Deploy(context.Background(), spec)
			if err != nil {
				panic(err)
			}

			if code != http.StatusOK && code != http.StatusAccepted {
				panic(err)
			}
			log.Printf("Status: %d", code)

		} else if action == "delete" {
			log.Printf("Deleting function %d: %s", i+1, name)

			namespace := "openfaas-fn"
			if err := client.DeleteFunction(context.Background(), name, namespace); err != nil {
				panic(err)
			}

		}

	}
}

func lookupPasswordViaKubectl() string {

	cmd := execute.ExecTask{
		Command:      "kubectl",
		Args:         []string{"get", "secret", "-n", "openfaas", "basic-auth", "-o", "jsonpath='{.data.basic-auth-password}'"},
		StreamStdio:  false,
		PrintCommand: false,
		Env:          os.Environ(),
	}

	res, err := cmd.Execute(context.Background())
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
