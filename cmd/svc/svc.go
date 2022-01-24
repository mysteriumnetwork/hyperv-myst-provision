package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

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
	"github.com/mysteriumnetwork/hyperv-node/service/logconfig"
	"github.com/mysteriumnetwork/hyperv-node/service/platform"
	"github.com/mysteriumnetwork/hyperv-node/service/util"
	"github.com/mysteriumnetwork/hyperv-node/service/util/winutil"
	"github.com/mysteriumnetwork/myst-launcher/utils"
)

func main() {
	defer util.PanicHandler("main")
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
		path, err := util.ThisPath()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to determine MysteriumVMSvc path")
		}
		options := install.Options{
			ExecuatblePath: path,
		}
		log.Info().Msgf("Installing supervisor with options: %#v", options)
		if err = install.Install(options); err != nil {
			log.Fatal().Err(err).Msg("Failed to install MysteriumVMSvc")
		}
		log.Info().Msg("Supervisor installed")

	} else if *flags.FlagUninstall {
		disableVM()

		log.Info().Msg("Uninstalling MysteriumVMSvc")
		if err := install.Uninstall(); err != nil {
			log.Fatal().Err(err).Msg("Failed to uninstall MysteriumVMSvc")
		}
		log.Info().Msg("MysteriumVMSvc uninstalled")

	} else if *flags.FlagImportVM {
		enableVM(*flags.FlagImportVMPreferEthernet)

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
			//n := winutil.NewNotifier()
			//n.WaitForIPChange()
			//return

			platformMgr, _ := platform.NewManager()
			ok, err := platformMgr.Features()
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to determine HyperV")
			}
			if !ok {
				log.Info().Msg("HyperV is not enabled")
				err := platformMgr.EnableHyperVPlatform()
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to enable HyperV")
				}
			}

			//err = func() error {
			//	m, err := hyperv_wmi.NewVMManager("Myst HyperV Alpine")
			//	if err != nil {
			//		return errors.Wrap(err, "NewVMManager")
			//	}
			//
			//	// find external ethernet port and get its device eepPath
			//	eep, adp, devID, err := m.FindDefaultNetworkAdapter(false)
			//	if err != nil {
			//		return errors.Wrap(err, "FindDefaultNetworkAdapter")
			//	}
			//	eepPath, err := eep.Path()
			//	if err != nil {
			//		return errors.Wrap(err, "Path")
			//	}
			//	fmt.Println(eepPath, devID)
			//
			//	//WaitForNetworkReady
			//	m.AdapterHasIPAddress(adp)
			//
			//	return nil
			//}()
			//fmt.Println(err)
			//return

			for {
				fmt.Println("")
				fmt.Println("----------------------------------------------")
				fmt.Println("[1] Enable node VM (use Ethernet connection)")
				fmt.Println("[2] Enable node VM (use Wifi connection)")
				fmt.Println("[3] Disable node VM")
				fmt.Println("[4] Exit")
				fmt.Print("\n> ")
				b, _ := bufio.NewReader(os.Stdin).ReadBytes('\n')
				k := strings.TrimSuffix(string(b), "\r\n")
				switch k {
				case "1", "2":
					path, err := util.ThisPath()
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to determine MysteriumVMSvc path")
					}
					options := install.Options{
						ExecuatblePath: path,
					}
					log.Info().Msgf("Installing supervisor with options: %#v", options)
					if err = install.Install(options); err != nil {
						log.Fatal().Err(err).Msg("Failed to install MysteriumVMSvc")
					}
					log.Info().Msg("MysteriumVMSvc installed")
					enableVM(k == "1")

				case "3":
					disableVM()

					log.Info().Msg("Uninstalling MysteriumVMSvc")
					if err := install.Uninstall(); err != nil {
						log.Fatal().Err(err).Msg("Failed to uninstall MysteriumVMSvc")
					}
					log.Info().Msg("MysteriumVMSvc uninstalled")

				case "4":
					return
				}
			}
		}
	}
}

func enableVM(preferEthernet bool) {
	var conn net.Conn
	err := utils.Retry(3, time.Second, func() error {
		var err error
		conn, err = winio.DialPipe(consts.Sock, nil)
		return err
	})
	if err != nil {
		log.Err(err).Msg("error listening")
		return
	}
	defer conn.Close()

	homeDir, err := windows.KnownFolderPath(windows.FOLDERID_Profile, windows.KF_FLAG_CREATE)
	if err != nil {
		log.Err(err).Msg("error getting profile path")
		return
	}
	keystorePath := homeDir + `\.mysterium\keystore`
	cmd := hyperv_wmi.KVMap{
		"cmd":             "import-vm",
		"keystore":        keystorePath,
		"report-progress": true,
		"prefer-ethernet": preferEthernet,
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
}

func disableVM() {
	var conn net.Conn
	err := utils.Retry(3, time.Second, func() error {
		var err error
		conn, err = winio.DialPipe(consts.Sock, nil)
		return err
	})
	if err != nil {
		log.Print("error listening")
		return
	}
	defer conn.Close()

	cmd := hyperv_wmi.KVMap{
		"cmd": "stop-vm",
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
		return
	}
}
