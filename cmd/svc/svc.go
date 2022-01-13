package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mysteriumnetwork/myst-launcher/utils"

	"github.com/mysteriumnetwork/hyperv-node/service/logconfig"

	"github.com/Microsoft/go-winio"
	"github.com/gonutz/w32"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"

	consts "github.com/mysteriumnetwork/hyperv-node/const"
	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/client"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/service/install"
	"github.com/mysteriumnetwork/hyperv-node/service/util"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
)

func main() {
	util.PanicHandler("main")
	flags.Parse()

	workDir, err := winutil.AppDataDir()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting AppDataDir: " + err.Error())
	}
	err = os.Chdir(workDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to configure logging")
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
		mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
		if err != nil {
			log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
		}

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

		fmt.Println("flags.FlagImportVMPreferEthernet", *flags.FlagImportVMPreferEthernet)
		cmd := hyperv_wmi.KVMap{
			"cmd":             "import-vm",
			"keystore":        keystorePath,
			"report-progress": true,
			"prefer-ethernet": *flags.FlagImportVMPreferEthernet,
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

		data := hyperv_wmi.NewKVMap(kv["data"])
		if data != nil {
			ip, ok := data["NetworkAddressIPv4"].(string)
			if ok && ip != "" {
				log.Print("Web UI is at http://" + ip + ":4449")
				fmt.Println("Web UI is at http://" + ip + ":4449")

				time.Sleep(7 * time.Second)
				util.OpenUrlInBrowser("http://" + ip + ":4449")
				return
			}
		}

	} else if *flags.FlagWinService {
		mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
		if err != nil {
			log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
		}
		logOpts := logconfig.LogOptions{
			LogLevel: "info",
			Filepath: "",
		}
		if err := logconfig.Configure(logOpts); err != nil {
			log.Fatal().Err(err).Msg("Failed to configure logging")
		}

		// Start service
		svc := daemon.New(mgr)
		if err := svc.Start(transport.Options{WinService: *flags.FlagWinService}); err != nil {
			log.Fatal().Err(err).Msg("Error running MysteriumVMSvc")
		}
	} else {

		if !w32.SHIsUserAnAdmin() {
			utils.RunasWithArgsNoWait("")
			return
		} else {

			for {
				fmt.Println("")
				fmt.Println("---------------------")
				fmt.Println("[1] Enable node VM (prefer Ethernet connection)")
				fmt.Println("[1] Enable node VM (prefer connection)")
				fmt.Println("[2] Disable node VM")
				fmt.Println("")
				fmt.Print("?>")
				b, _ := bufio.NewReader(os.Stdin).ReadBytes('\n')

				switch strings.TrimSuffix(string(b), "\r\n") {
				case "1":
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

				case "2":
					mgr, err := hyperv_wmi.NewVMManager(*flags.FlagVMName)
					if err != nil {
						log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
					}

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

				}

			}
			//path, err := thisPath()
			//if err != nil {
			//	log.Fatal().Err(err).Msg("Failed to determine MysteriumVMSvc path")
			//}
			//options := install.Options{
			//	SupervisorPath: path,
			//}
			//log.Info().Msgf("Installing supervisor with options: %#v", options)
			//if err = install.Install(options); err != nil {
			//	log.Fatal().Err(err).Msg("Failed to install MysteriumVMSvc")
			//}
			//log.Info().Msg("Supervisor installed")

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
