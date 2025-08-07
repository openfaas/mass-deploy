package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/openfaas/go-sdk"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mass-deploy",
	Short: "Mass deploy functions to OpenFaaS",
	Long: `Mass deploy functions to OpenFaaS with support for parallel deployment
and management of multiple functions.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.String("gateway", "http://127.0.0.1:8080", "gateway url")
	flags.String("namespace", "openfaas-fn", "namespace for functions")
	flags.Int("workers", 1, "number of workers to use")

	rootCmd.AddCommand(makeCreateCmd())
	rootCmd.AddCommand(makeDeleteCmd())
	rootCmd.AddCommand(makeInvokeCmd())
}

func getClient(cmd *cobra.Command) (*sdk.Client, error) {
	gateway, _ := cmd.Flags().GetString("gateway")
	gatewayURL, err := url.Parse(gateway)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway URL: %w", err)
	}

	auth := &sdk.BasicAuth{}

	if gatewayURL.User != nil {
		auth.Username = gatewayURL.User.Username()
		auth.Password, _ = gatewayURL.User.Password()
	} else {
		auth.Username = "admin"
		auth.Password = lookupPasswordViaKubectl()
	}

	return sdk.NewClient(gatewayURL, auth, http.DefaultClient), nil
}
