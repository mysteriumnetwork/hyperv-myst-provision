package main

import (
	"fmt"
	"log"

	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

var shell = powershell.New(powershell.OptionDebugPrint)

func main() {

	_, err := shell.Execute("Get-Module -listavailable")
	if err != nil {
		fmt.Println("ERROR!!!")
		log.Fatal(err)
		return
	}

	_, err = shell.Execute("Get-VM")
	if err != nil {
		fmt.Println("ERROR!!!")
		log.Fatal(err)
	}
}
