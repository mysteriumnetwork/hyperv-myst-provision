package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/gonutz/w32"
	"github.com/pkg/errors"
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
		conn, err := connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Connect")
		}
		defer conn.Close()
		disableVM(conn)

		log.Info().Msg("Uninstalling MysteriumVMSvc")
		if err := install.Uninstall(); err != nil {
			log.Fatal().Err(err).Msg("Failed to uninstall MysteriumVMSvc")
		}
		log.Info().Msg("MysteriumVMSvc uninstalled")

	} else if *flags.FlagImportVM {
		conn, err := connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Connect")
		}
		defer conn.Close()

		enableVM(conn, *flags.FlagImportVMPreferEthernet, "")

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
			//m, err := hyperv_wmi.NewVMManager("Myst HyperV Alpine")
			//if err != nil {
			//	fmt.Println(err)
			//	return
			//}
			//m.SelectAdapter()
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

			for {
				fmt.Println("")
				fmt.Println("Select action")
				fmt.Println("----------------------------------------------")
				fmt.Println("1  Enable node VM (use Ethernet connection)")
				fmt.Println("2  Enable node VM (use Wifi connection)")
				fmt.Println("3  Enable node VM (select adapter manually)")
				fmt.Println("4  Disable node VM")
				fmt.Println("5  Exit")
				fmt.Print("\n> ")
				k := util.ReadConsole()
				if k == "5" {
					return
				}

				var conn net.Conn
				switch k {
				case "1", "2", "3", "4":
					err = installSvc()
					if err != nil {
						log.Fatal().Err(err).Msg("Install service")
					}
					conn, err = connect()
					if err != nil {
						log.Fatal().Err(err).Msg("Connect")
					}
				}

				switch k {
				case "1", "2":
					err = enableVM(conn, k == "1", "")
					if err != nil {
						log.Fatal().Err(err).Msg("Enable VM")
					}

				case "3":
					ID, err := selectAdapter(conn)
					if err != nil {
						log.Fatal().Err(err).Msg("Select adapter")
					}
					err = enableVM(conn, false, ID)
					if err != nil {
						log.Fatal().Err(err).Msg("Enable VM")
					}

				case "4":
					disableVM(conn)

				case "5":
					return
				}
			}
		}
	}
}

func connect() (net.Conn, error) {
	var conn net.Conn
	err := utils.Retry(3, time.Second, func() error {
		var err error
		conn, err = winio.DialPipe(consts.Sock, nil)
		return err
	})
	if err != nil {
		log.Err(err).Msg("error listening")
		return nil, err
	}
	return conn, nil
}

func installSvc() error {
	path, err := util.ThisPath()
	if err != nil {
		return errors.Wrap(err, "Failed to determine MysteriumVMSvc path")
	}
	options := install.Options{
		ExecuatblePath: path,
	}
	log.Info().Msgf("Installing supervisor with options: %#v", options)
	if err = install.Install(options); err != nil {
		return errors.Wrap(err, "Failed to install MysteriumVMSvc")
	}
	log.Info().Msg("MysteriumVMSvc installed")
	return nil
}

func enableVM(conn net.Conn, preferEthernet bool, ID string) error {

	homeDir, err := windows.KnownFolderPath(windows.FOLDERID_Profile, windows.KF_FLAG_CREATE)
	if err != nil {
		log.Err(err).Msg("error getting profile path")
		return err
	}
	keystorePath := homeDir + `\.mysterium\keystore`
	cmd := hyperv_wmi.KVMap{
		"cmd":             daemon.CommandImportVM,
		"keystore":        keystorePath,
		"report-progress": true,
		"prefer-ethernet": preferEthernet,
		"adapter-id":      ID,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		log.Error().Msgf("Send command: %s", res["err"])
		return err
	}

	cmd = hyperv_wmi.KVMap{
		"cmd": daemon.CommandGetKvp,
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
			return nil
		}
	}

	return nil
}

func disableVM(conn net.Conn) {
	cmd := hyperv_wmi.KVMap{
		"cmd": daemon.CommandStopVM,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
		return
	}
}

func selectAdapter(conn net.Conn) (string, error) {
	cmd := hyperv_wmi.KVMap{
		"cmd": daemon.CommandGetAdapters,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
		return "", errors.New(fmt.Sprint("error:", res["err"]))
	}

	fmt.Println("Select adapter")
	fmt.Println("----------------------------------------------")
	l := res["data"].([]interface{})
	for k, v := range l {
		fmt.Println(k+1, "", v.(map[string]interface{})["Name"])
	}
	fmt.Print("\n> ")
	k_ := util.ReadConsole()
	k, err := strconv.ParseInt(k_, 10, 8)
	if err != nil {
		log.Err(err).Msg("ParseInt error")
		return "", err
	}
	if k < 0 || k > int64(len(l)) {
		log.Err(err).Msg("Wrong number")
		return "", errors.New("Wrong number")
	}
	ID := l[k-1].(map[string]interface{})["ID"].(string)
	fmt.Println(k, ID)
	return ID, nil
}
