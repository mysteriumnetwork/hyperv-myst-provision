package main

import (
	"log"
	"os"

	"github.com/mysteriumnetwork/hyperv-node/vm-agent/server"

	"github.com/sevlyar/go-daemon"
)

func main() {
	cntxt := &daemon.Context{
		PidFileName: "/run/vm-myst-agent.pid",
		PidFilePerm: 0644,
		LogFileName: "vm-myst-agent.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[vm-myst-agent]"},
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Print("daemon started >")
	log.Print(os.Args)
	server.Serve()
	log.Println("daemon exit >")
}
