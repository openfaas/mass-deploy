package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/openfaas/go-sdk"
	"github.com/spf13/cobra"
)

func makeDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete multiple functions",
		Long: `Delete multiple functions in parallel.
Each function will be deleted based on the name pattern with numeric suffix.`,
		RunE: runDelete,
	}

	flags := cmd.Flags()
	flags.String("name", "env", "name of the function")
	flags.Int("functions", 100, "number of functions to delete")
	flags.Int("start-at", 0, "start at function number")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	functions, _ := cmd.Flags().GetInt("functions")
	startAt, _ := cmd.Flags().GetInt("start-at")
	namespace, _ := cmd.Flags().GetString("namespace")
	workers, _ := cmd.Flags().GetInt("workers")

	wg := sync.WaitGroup{}
	wg.Add(workers)

	workChan := make(chan string)
	started := time.Now()

	for i := 0; i < workers; i++ {
		go func(worker int) {
			for name := range workChan {
				if len(name) > 0 {
					if err := deleteFunction(worker, name, client, namespace); err != nil {
						log.Printf("[%d] Error deleting %s: %s", worker, name, err)
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

func deleteFunction(worker int, name string, client *sdk.Client, namespace string) error {
	start := time.Now()
	log.Printf("[%d] Deleting function: %s", worker, name)

	if err := client.DeleteFunction(context.Background(), name, namespace); err != nil {
		log.Printf("[%d] Delete %s, error: %s", worker, name, err)

		if !strings.Contains(err.Error(), "not found") {
			return err
		}
	}
	log.Printf("[%d] Deleted function: %s (%dms)", worker, name, time.Since(start).Milliseconds())
	return nil
}
