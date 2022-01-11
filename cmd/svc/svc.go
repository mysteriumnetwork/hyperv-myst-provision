package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Microsoft/go-winio"
	consts "github.com/mysteriumnetwork/hyperv-node/const"
	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/client"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/service/install"
	"github.com/mysteriumnetwork/hyperv-node/service/logconfig"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
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
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to configure logging")
	}
	mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
	if err != nil {
		log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
	}

	if *flags.FlagInstall {
		path, err := thisPath()
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

	} else if *flags.FlagImportVM {

		homeDir, err := windows.KnownFolderPath(windows.FOLDERID_Profile, windows.KF_FLAG_CREATE)
		if err != nil {
			log.Err(err).Msg("error getting profile path")
			return
		}
		keystorePath := homeDir + `\.mysterium\keystore`

		conn, err := winio.DialPipe(consts.Sock, nil)
		if err != nil {
			log.Err(err).Msg("error listening")
			return
		}
		defer conn.Close()

		cmd := hyperv_wmi.KVMap{
			"cmd":             "import-vm",
			"keystore":        keystorePath,
			"report-progress": true,
		}
		res := client.SendCommand(conn, cmd)
		if res["resp"] == "error" {
			fmt.Println("error:", res["err"])
			return
		}

		cmd = hyperv_wmi.KVMap{
			"cmd": "get-kvp",
		}
		kv := client.SendCommand(conn, cmd)
		log.Info().Msgf("KV: %v", kv)

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
