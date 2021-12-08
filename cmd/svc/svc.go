package main

import (
	"github.com/mysteriumnetwork/hyperv-node/daemon"
	"github.com/mysteriumnetwork/hyperv-node/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/rs/zerolog/log"
)

func main() {
	flags.Parse()

	mgr, _ := hyperv_wmi.NewVMManager(*flags.FlagVMName)

	supervisor := daemon.New(mgr)
	if err := supervisor.Start(transport.Options{WinService: *flags.FlagWinService}); err != nil {
		log.Fatal().Err(err).Msg("Error running supervisor")
	}
}
