// Copyright (c) 2023 Alex Ellis, OpenFaaS Ltd
// License: MIT

package main

import (
	"log"

	"github.com/alexellis/mass-deploy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
