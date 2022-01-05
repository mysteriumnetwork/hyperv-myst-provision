package main

import (
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/service/install"
	"github.com/mysteriumnetwork/hyperv-node/service/logconfig"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

func main() {
	flags.Parse()

	logOpts := logconfig.LogOptions{
		LogLevel: "info",
		Filepath: "",
	}
	if err := logconfig.Configure(logOpts); err != nil {
		log.Fatal().Err(err).Msg("Failed to configure logging")
	}

	workDir, err := winutil.AppDataDir()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting AppDataDir: " + err.Error())
	}
	err = os.Chdir(workDir)
	log.Err(err)

	mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
	if err != nil {
		log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
	}

	if *flags.FlagInstall {
		path, err := thisPath()
		fmt.Println("path", path)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to determine MysteriumVMSvc path")
		}
		options := install.Options{
			SupervisorPath: path,
		}
		log.Info().Msgf("Installing supervisor with options: %#v", options)
		if err = install.Install(options); err != nil {
			log.Fatal().Err(err).Msg("Failed to install MysteriumVMSvc")
		}
		log.Info().Msg("Supervisor installed")

	} else if *flags.FlagUninstall {
		log.Info().Msg("Uninstalling MysteriumVMSvc")
		if err := install.Uninstall(); err != nil {
			log.Fatal().Err(err).Msg("Failed to uninstall MysteriumVMSvc")
		}
		if err := mgr.StopVM(); err != nil {
			log.Fatal().Err(err).Msg("Failed to stop VM")
		}
		if err := mgr.RemoveVM(); err != nil {
			log.Fatal().Err(err).Msg("Failed to remove VM")
		}
		log.Info().Msg("MysteriumVMSvc uninstalled")

	} else {
		// Start service
		svc := daemon.New(mgr)
		if err := svc.Start(transport.Options{WinService: *flags.FlagWinService}); err != nil {
			log.Fatal().Err(err).Msg("Error running MysteriumVMSvc")
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
