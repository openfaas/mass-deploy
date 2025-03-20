package cmd

import (
	"context"
	b64 "encoding/base64"
	"os"
	"strings"

	"github.com/alexellis/go-execute/v2"
)

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
