package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/gonutz/w32"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"

	consts "github.com/mysteriumnetwork/hyperv-node/const"
	"github.com/mysteriumnetwork/hyperv-node/model"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon/client"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon/flags"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/service2/install"
	"github.com/mysteriumnetwork/hyperv-node/service2/logconfig"
	"github.com/mysteriumnetwork/hyperv-node/service2/platform"
	"github.com/mysteriumnetwork/hyperv-node/service2/util"
	"github.com/mysteriumnetwork/hyperv-node/service2/util/winutil"
	"github.com/mysteriumnetwork/hyperv-node/vbox"
	"github.com/mysteriumnetwork/myst-launcher/utils"
)

func main() {
	defer util.PanicHandler("main")
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

	if *flags.FlagInstall {
		// TODO: check admin rights before
		platformMgr, _ := platform.NewManager()
		err = platformMgr.EnableVirtualBox()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to enable VirtualBox")
		}

		installSvc()

	} else if *flags.FlagUninstall {
		conn, err := connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Connect")
		}
		defer conn.Close()
		disableVM(conn)

		uninstallSvc()

	} else if *flags.FlagImportVM {
		conn, err := connect()
		if err != nil {
			log.Fatal().Err(err).Msg("Connect")
		}
		defer conn.Close()

		importVM(conn)

	} else if *flags.FlagWinService {

		cfg := new(model.Config)
		cfg.Read()

		mgr, err := vbox.NewVMManager(*flags.FlagVMName, cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("Error NewVMManager: " + err.Error())
		}
		logOpts := logconfig.LogOptions{
			LogLevel: "debug",
			Filepath: "",
		}
		if err := logconfig.Configure(logOpts); err != nil {
			log.Fatal().Err(err).Msg("Failed to configure logging")
		}

		// Start service
		svc := daemon.New(mgr, cfg)
		if err := svc.Start(transport.Options{WinService: *flags.FlagWinService}); err != nil {
			log.Fatal().Err(err).Msg("Error running MysteriumVMSvc")
		}

	} else {

		if !w32.SHIsUserAnAdmin() {
			utils.RunasWithArgsNoWait("")
			return
		} else {
			platformMgr, _ := platform.NewManager()
			err = platformMgr.EnableVirtualBox()
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to enable VirtualBox")
			}

			for {
				fmt.Println("Select an action")
				fmt.Println("----------------------------------------------")
				fmt.Println("1  Install DualMode service")
				fmt.Println("2  Uninstall DualMode service")
				fmt.Println("3  Import node VM")
				fmt.Println("4  Enable node VM")
				fmt.Println("5  Disable node VM")
				fmt.Println("6  Update node")
				fmt.Println("")
				fmt.Println("7  Exit")
				fmt.Print("\n> ")
				k := util.ReadConsole()

				var conn net.Conn
				switch k {
				case "3", "4", "5", "6":
					homeDir, _ := os.UserHomeDir()
					keystorePath := path.Join(homeDir, consts.KeystorePath)

					if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
						log.Info().Msg("Keystore not found")
						continue
					}
					conn, err = connect()
					if err != nil {
						log.Fatal().Err(err).Msg("Connect")
					}
				}

				switch k {
				case "1":
					installSvc()
				case "2":
					uninstallSvc()

				case "3":
					err = importVM(conn)
					if err != nil {
						log.Fatal().Err(err).Msg("Import VM")
					}

				case "4":
					err = enableVM(conn)
					if err != nil {
						log.Fatal().Err(err).Msg("Enable VM")
					}

				case "5":
					disableVM(conn)

				case "6":
					updateNode(conn)

				case "7":
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
	log.Info().Msgf("Installing dual-mode helper with options: %#v", options)
	if err = install.Install(options); err != nil {
		return errors.Wrap(err, "Failed to install MysteriumVMSvc")
	}
	log.Info().Msg("MysteriumVMSvc installed")
	return nil
}

func uninstallSvc() error {
	log.Info().Msgf("Installing dual-mode helper")
	if err := install.Uninstall(); err != nil {
		return errors.Wrap(err, "Failed to uninstall MysteriumVMSvc")
	}
	log.Info().Msg("MysteriumVMSvc uninstalled")
	return nil
}

func importVM(conn net.Conn) error {
	keystorePath, err := getKeystorePath()
	if err != nil {
		return err
	}

	cmd := model.KVMap{
		"cmd":             daemon.CommandImportVM,
		"keystore":        keystorePath,
		"report-progress": true,

		//"prefer-ethernet": preferEthernet,
		//"adapter-id":      ID,
		//"adapter-name":    Name,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		log.Error().Msgf("Send command: %s", res["err"])
		return err
	}
	fmt.Println()

	return nil
}

func getKeystorePath() (string, error) {
	homeDir, err := windows.KnownFolderPath(windows.FOLDERID_Profile, windows.KF_FLAG_CREATE)
	if err != nil {
		log.Err(err).Msg("error getting profile path")
		return "", err
	}
	keystorePath := path.Join(homeDir, consts.KeystorePath)
	return keystorePath, nil
}

func enableVM(conn net.Conn) error {
	keystorePath, err := getKeystorePath()
	if err != nil {
		return err
	}

	cmd := model.KVMap{
		"cmd":      daemon.CommandStartVM,
		"keystore": keystorePath,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		log.Error().Msgf("Send command: %s", res["err"])
		return err
	}

	dataStr, _ := json.Marshal(res["data"])
	fmt.Println()
	fmt.Println("Report:", string(dataStr))

	cmd = model.KVMap{
		"cmd": daemon.CommandGetKvp,
	}
	kv := client.SendCommand(conn, cmd)
	fmt.Println()

	data := model.NewKVMap(kv["data"])
	if data != nil {

		ip, ok := data[vbox.KeyIP].(string)
		if ok && ip != "" {
			log.Print("Web UI is at http://" + ip + ":4449")

			time.Sleep(1 * time.Second)
			util.OpenUrlInBrowser("http://" + ip + ":4449")
			return nil
		}
	}

	return nil
}

func disableVM(conn net.Conn) {
	cmd := model.KVMap{
		"cmd": daemon.CommandStopVM,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
		return
	}
}

func updateNode(conn net.Conn) {
	fmt.Println("updateNode")

	cmd := model.KVMap{
		"cmd": daemon.CommandUpdateNode,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
	}
	fmt.Println("updateNode", res)

}

//returns: adapter ID, Name
func selectAdapter(conn net.Conn) (string, string, error) {
	cmd := model.KVMap{
		"cmd": daemon.CommandGetAdapters,
	}
	res := client.SendCommand(conn, cmd)
	if res["resp"] == "error" {
		fmt.Println("error:", res["err"])
		return "", "", errors.New(fmt.Sprint("error:", res["err"]))
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
		return "", "", err
	}
	if k < 0 || k > int64(len(l)) {
		log.Err(err).Msg("Wrong number")
		return "", "", errors.New("Wrong number")
	}
	adapter := l[k-1].(map[string]interface{})

	ID := adapter["ID"].(string)
	Name := adapter["Name"].(string)
	return ID, Name, nil
}
