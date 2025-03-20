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

var (
	deleteName      string
	deleteFunctions int
	deleteStartAt   int
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete multiple functions",
	Long: `Delete multiple functions in parallel.
Each function will be deleted based on the name pattern with numeric suffix.`,
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVar(&deleteName, "name", "env", "name of the function")
	deleteCmd.Flags().IntVar(&deleteFunctions, "functions", 100, "number of functions to delete")
	deleteCmd.Flags().IntVar(&deleteStartAt, "start-at", 0, "start at function number")
}

func runDelete(cmd *cobra.Command, args []string) error {
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
					if err := deleteFunction(worker, name, client, namespace); err != nil {
						log.Printf("[%d] Error deleting %s: %s", worker, name, err)
					}
				}
			}
			wg.Done()
		}(i)
	}

	for i := deleteStartAt; i < deleteFunctions; i++ {
		functionName := fmt.Sprintf("%s-%d", deleteName, i+1)
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
