package main

import (
	"github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/service/install"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

func main() {
	flags.Parse()
	mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
	if err != nil {
		log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
	}

	if *flags.FlagInstall {
		path, err := thisPath()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to determine supervisor's path")
		}
		options := install.Options{
			SupervisorPath: path,
		}
		log.Info().Msgf("Installing supervisor with options: %#v", options)
		if err = install.Install(options); err != nil {
			log.Fatal().Err(err).Msg("Failed to install supervisor")
		}
		log.Info().Msg("Supervisor installed")
	} else {
		supervisor := daemon.New(mgr)
		workDir, err := winutil.AppDataDir()
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting AppDataDir: " + err.Error())
		}
		os.Chdir(workDir)

		if err := supervisor.Start(transport.Options{WinService: *flags.FlagWinService}); err != nil {
			log.Fatal().Err(err).Msg("Error running supervisor")
		}
	}
}

func thisPath() (string, error) {
	thisExec, err := os.Executable()
	if err != nil {
		return "", err
	}
	thisPath, err := filepath.Abs(thisExec)
	if err != nil {
		return "", err
	}
	return thisPath, nil
}
