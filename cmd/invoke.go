package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openfaas/go-sdk"
	"github.com/spf13/cobra"
)

func makeInvokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Invoke multiple functions",
		Long:  "Invoke multiple functions in parallel",
		RunE:  runInvoke,
	}

	flag := cmd.Flags()
	flag.Int("functions", 100, "number of functions to create")
	flag.Int("start-at", 0, "start at function number")
	flag.String("name", "env", "name of function")
	flag.Bool("async", false, "Invoke use asynchronously")
	flag.Duration("deadline", time.Second*120, "Deadline for the invocation")
	flag.IntP("requests", "n", 1, "Number of requests to send per function invocation")

	return cmd
}

func runInvoke(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	name, _ := cmd.Flags().GetString("name")
	functions, _ := cmd.Flags().GetInt("functions")
	startAt, _ := cmd.Flags().GetInt("start-at")
	workers, _ := cmd.Flags().GetInt("workers")
	async, _ := cmd.Flags().GetBool("async")
	deadline, _ := cmd.Flags().GetDuration("deadline")
	requests, _ := cmd.Flags().GetInt("n")

	wg := sync.WaitGroup{}
	wg.Add(workers)

	started := time.Now()

	workChan := make(chan string)

	for i := 0; i < workers; i++ {
		go func(worker int) {
			for name := range workChan {
				if len(name) > 0 {
					for j := 0; j < requests; j++ {
						if err := invoke(worker, j+1, requests, name, namespace, client, async, deadline); err != nil {
							log.Print(err)
						}
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
	return nil
}

func invoke(worker, reqNo, requests int, name, namespace string, client *sdk.Client, async bool, deadline time.Duration) error {
	start := time.Now()
	log.Printf("[%d] %d/%d Invoking function: %s", worker, reqNo, requests, name)

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", nil)
	if err != nil {
		return err
	}

	auth := false
	res, err := client.InvokeFunction(name, namespace, async, auth, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	log.Printf("[%d] %d/%d Invoked function: %s [%d] (%dms)", worker, reqNo, requests, name, res.StatusCode, time.Since(start).Milliseconds())

	return nil
}
