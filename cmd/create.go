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

func makeCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create multiple functions",
		Long: `Create multiple functions in parallel with the specified configuration.
Each function will be named with a numeric suffix.`,
		RunE: runCreate,
	}

	flags := cmd.Flags()
	flags.String("name", "env", "name of the function")
	flags.String("image", "", "image to use for the function")
	flags.String("fprocess", "env", "fprocess to use for the function")
	flags.Int("functions", 100, "number of functions to create")
	flags.Int("start-at", 0, "start at function number")
	flags.StringArray("env", []string{}, "environment variables to set (format: KEY=VALUE)")
	flags.StringArray("label", []string{}, "labels to set on the function (format: KEY=VALUE)")
	flags.Bool("update-existing", false, "update existing functions, when set to false, any existing functions are skipped")

	cmd.MarkFlagRequired("image")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	image, _ := cmd.Flags().GetString("image")
	fprocess, _ := cmd.Flags().GetString("fprocess")
	functions, _ := cmd.Flags().GetInt("functions")
	startAt, _ := cmd.Flags().GetInt("start-at")
	envVars, _ := cmd.Flags().GetStringArray("env")
	labels, _ := cmd.Flags().GetStringArray("label")
	namespace, _ := cmd.Flags().GetString("namespace")
	workers, _ := cmd.Flags().GetInt("workers")
	updateExisting, _ := cmd.Flags().GetBool("update-existing")

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
