package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/itzg/go-flagsfiller"

	"github.com/mysteriumnetwork/hyperv-node/provisioner"
)

type flagsSet struct {
	NodeVersion string `usage:"node version to be installed on guest VM, if no specified latest will be used"`
}

func (fs *flagsSet) validate() error {
	return nil
}

func flagsParse() {
	if err := flagsfiller.New().Fill(flag.CommandLine, &flags); err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	if err := flags.validate(); err != nil {
		log.Fatal(err)
	}
}

var flags flagsSet

func main() {
	flagsParse()
	ver, err := provisioner.GetLatestNodeVersion(flags.NodeVersion)
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(ver)
}
