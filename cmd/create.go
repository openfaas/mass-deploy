package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/go-sdk"
	"github.com/spf13/cobra"
)

var (
	name      string
	image     string
	fprocess  string
	functions int
	startAt   int
	envVars   []string
	labels    []string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create multiple functions",
	Long: `Create multiple functions in parallel with the specified configuration.
Each function will be named with a numeric suffix.`,
	RunE: runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVar(&name, "name", "env", "name of the function")
	createCmd.Flags().StringVar(&image, "image", "", "image to use for the function")
	createCmd.Flags().StringVar(&fprocess, "fprocess", "env", "fprocess to use for the function")
	createCmd.Flags().IntVar(&functions, "functions", 100, "number of functions to create")
	createCmd.Flags().IntVar(&startAt, "start-at", 0, "start at function number")
	createCmd.Flags().StringArrayVar(&envVars, "env", []string{}, "environment variables to set (format: KEY=VALUE)")
	createCmd.Flags().StringArrayVar(&labels, "label", []string{}, "labels to set on the function (format: KEY=VALUE)")
	createCmd.Flags().BoolVar(&updateExisting, "update-existing", false, "update existing functions")

	createCmd.MarkFlagRequired("image")
}

func runCreate(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(workers)

	workChan := make(chan string)
	started := time.Now()

	for i := 0; i < workers; i++ {
		go func(worker int) {
			for name := range workChan {
				if len(name) > 0 {
					if err := reconcile(worker, name, image, fprocess, client, namespace, "create", envVars, labels, updateExisting); err != nil {
						log.Printf("[%d] Error reconciling %s: %s", worker, name, err)
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

func reconcile(worker int, name, image, fprocess string, client *sdk.Client, namespace, action string, envVars, labels []string, updateExisting bool) error {
	if action == "create" {
		spec := types.FunctionDeployment{
			Service:   name,
			Image:     image,
			Namespace: namespace,
		}

		// Process environment variables
		envMap := make(map[string]string)
		for _, env := range envVars {
			if key, value, found := strings.Cut(env, "="); found {
				envMap[key] = value
			}
		}
		if len(envMap) > 0 {
			spec.EnvVars = envMap
		}

		// Process labels
		labelMap := make(map[string]string)
		for _, label := range labels {
			if key, value, found := strings.Cut(label, "="); found {
				labelMap[key] = value
			}
		}
		if len(labelMap) > 0 {
			spec.Labels = &labelMap
		}

		if len(fprocess) > 0 {
			spec.EnvProcess = fprocess
		}

		start := time.Now()

		update := false
		if _, err := client.GetFunction(context.Background(), name, namespace); err == nil {
			update = true
		}
		if update && !updateExisting {
			log.Printf("[%d] Function %s skipped", worker, name)
			return nil
		}

		if update {
			log.Printf("[%d] Updating: %s", worker, name)
			code, err := client.Update(context.Background(), spec)
			if err != nil {
				return err
			}
			if code != http.StatusOK && code != http.StatusAccepted {
				return fmt.Errorf("unexpected status code: %d", code)
			}
			log.Printf("[%d] Updated: %s, status: %d (%dms)", worker, name, code, time.Since(start).Milliseconds())
		} else {
			log.Printf("[%d] Creating: %s", worker, name)
			code, err := client.Deploy(context.Background(), spec)
			if err != nil {
				return err
			}
			if code != http.StatusOK && code != http.StatusAccepted {
				return fmt.Errorf("unexpected status code: %d", code)
			}
			log.Printf("[%d] Created: %s, status: %d (%dms)", worker, name, code, time.Since(start).Milliseconds())
		}
	}

	return nil
}
