// Copyright (c) 2023 Alex Ellis, OpenFaaS Ltd
// License: MIT

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
	"sync"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/go-sdk"
)

func main() {

	var (
		gateway, namespace          string
		action                      string
		name, image, fprocess       string
		functions, startAt, workers int
	)

	flag.StringVar(&gateway, "gateway", "http://127.0.0.1:8080", "gateway url")
	flag.StringVar(&namespace, "namespace", "openfaas-fn", "namespace for functions")
	flag.IntVar(&functions, "functions", 100, "number of functions to create")
	flag.IntVar(&startAt, "start-at", 0, "start at function number")
	flag.StringVar(&name, "name", "env", "name of function")
	flag.StringVar(&image, "image", "", "image to use for function")
	flag.StringVar(&fprocess, "fprocess", "env", "fprocess to use for function")
	flag.StringVar(&action, "action", "create", "action to perform")
	flag.IntVar(&workers, "workers", 1, "number of workers to use")

	flag.Parse()

	gatewayURL, err := url.Parse(gateway)
	if err != nil {
		panic(err)
	}

	auth := &sdk.BasicAuth{}

	if gatewayURL.User != nil {
		auth.Username = gatewayURL.User.Username()
		auth.Password, _ = gatewayURL.User.Password()
	} else {
		auth.Username = "admin"
		auth.Password = lookupPasswordViaKubectl()
	}

	if len(image) == 0 {
		panic("-image is required")
	}

	client := sdk.NewClient(gatewayURL, auth, http.DefaultClient)

	wg := sync.WaitGroup{}
	wg.Add(workers)

	started := time.Now()

	workChan := make(chan string)

	for i := 0; i < workers; i++ {
		go func(worker int) {
			for name := range workChan {
				if len(name) > 0 {
					if err := reconcile(worker, name, image, fprocess, client, namespace, action); err != nil {
						panic(err)
					}
				}
			}
			wg.Done()
		}(i)
	}

	for i := startAt; i < functions; i++ {
		functionName := fmt.Sprintf("%s-%d", name, i+1)
		workChan <- functionName
	}

	close(workChan)

	wg.Wait()

	log.Printf("Took: %.2f", time.Since(started).Seconds())

}

func reconcile(worker int, name, image, fprocess string, client *sdk.Client, namespace, action string) error {
	if action == "create" {
		log.Printf("[%d] Creating: %s", worker, name)
		spec := types.FunctionDeployment{
			Service:   name,
			Image:     image,
			Namespace: namespace,
		}

		if len(fprocess) > 0 {
			spec.EnvProcess = fprocess
		}

		start := time.Now()

		code, err := client.Deploy(context.Background(), spec)
		if err != nil {
			return err
		}

		if code != http.StatusOK && code != http.StatusAccepted {
			return err
		}
		log.Printf("[%d] Created: %s, status: %d (%dms)", worker, name, code, time.Since(start).Milliseconds())

	} else if action == "delete" {
		start := time.Now()
		log.Printf("[%d] Deleting function: %s", worker, name)

		if err := client.DeleteFunction(context.Background(), name, namespace); err != nil {
			log.Printf("[%d] Delete %s, error: %s", worker, name, err)

			if !strings.Contains(err.Error(), "not found") {
				return err
			}
		}
		log.Printf("[%d] Deleted function: %s (%dms)", worker, name, time.Since(start).Milliseconds())
	}

	return nil
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
