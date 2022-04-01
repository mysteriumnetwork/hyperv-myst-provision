package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/mysteriumnetwork/hyperv-node/vm-agent/utils"
	"github.com/oklog/run"
)

var (
	pr      *utils.ProcessRunner
	version string
)

func getLocalAddresses(intName string) (res []net.IP) {

	ifs, err := net.Interfaces()
	if err != nil {
		fmt.Print(fmt.Errorf("getLocalAddresses: %+v\n", err.Error()))
		return
	}

	for _, i := range ifs {
		if i.Name != intName {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			log.Print(fmt.Errorf("getLocalAddresses: %+v\n", err.Error()))
			continue
		}

		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				res = append(res, v.IP)
			}
		}
		return
	}
	return
}

func readEnvMyst() string {
	f, err := os.Open("/.env.myst")
	if err != nil {
		log.Println("open error:", err)
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	for s.Scan() {
		line := s.Text()
		pp := strings.Split(line, "=")
		if len(pp) == 2 {
			k, v := pp[0], pp[1]
			if strings.ToLower(k) == "launcher" {
				return v
			}
		}
	}
	return ""
}

func saveEnvMyst() {
	txt := []byte(fmt.Sprintf("launcher=%s", version))
	err := os.WriteFile("/.env.myst", txt, 0644)
	if err != nil {
		log.Println("WriteFile", err)
	}
}

func setMystCmdArgs() {
	pr.SetArgs(binMyst, "--Keystore.lightweight", "--local-service-discovery=false", "--launcher.ver="+version, "service", "--agreed-terms-and-conditions")
}

func checkKeystore() bool {
	files, err := ioutil.ReadDir(Keystore)
	if err != nil {
		return false
	}
	return len(files) >= 2
}

/////////////////////////////////////////
type ActorWrap struct {
	pr *utils.ProcessRunner
}

func (a *ActorWrap) Start() error {
	// wait for keystotre
	for !checkKeystore() {
		time.Sleep(2 * time.Second)
	}
	return a.pr.Start()
}

func (a *ActorWrap) Stop() error {
	a.pr.Shutdown()
	return nil
}


func Serve() {
	version = readEnvMyst()
	log.Println("version >>>>", version)
	if version == "" {
		version = "0.0.0"
	}

	pr = utils.NewProcessRunner()
	setMystCmdArgs()

	l, err := vsock.Listen(30, nil)
	log.Println(l, err)

	http.HandleFunc("/", httpHandler)
	listener1, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Println(err)
		return
	}
	listener2, err := net.Listen("tcp", "0.0.0.0:8081")
	if err != nil {
		log.Println(err)
		return
	}
	srv := http.Server{}

	a := ActorWrap{pr}

	// Start & shutdown gracefully
	{
		var g run.Group
		s := utils.NewSigTermHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		g.Add(func() error { return s.Wait() }, func(err error) { s.Stop() })
		// g.Add(func() error { return pr.Start() }, func(err error) { pr.Shutdown() })
		g.Add(func() error { return a.Start() }, func(err error) { a.Stop() })
		g.Add(func() error { return srv.Serve(listener1) }, func(err error) { srv.Shutdown(ctx) })
		g.Add(func() error { return srv.Serve(listener2) }, func(err error) { srv.Shutdown(ctx) })

		g.Run()
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)

	switch r.URL.Path {
	case "/set":
		versionNew := r.URL.Query().Get("launcher")
		if versionNew != "" && version != versionNew {
			version = versionNew
			saveEnvMyst()

			setMystCmdArgs()
			pr.StopCommand()
			pr.StartCommand()
		}

	case "/stop":
		pr.StopCommand()

	case "/start":
		pr.StartCommand()

	case "/state":
		res := make(map[string]interface{})
		res["ips"] = getLocalAddresses(interface_)

		version := GetNodeVersion()
		res["version"] = version
		json.NewEncoder(w).Encode(res)

	case "/update":
		cmd := exec.Command("/bin/sh", "/root/update-myst.sh", runtime.GOARCH)
		if err := cmd.Start(); err != nil {
			log.Println("start", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			log.Println("start", err)
			return
		}
		pr.StopCommand()
		pr.StartCommand()

	case "/upload":
		err := r.ParseMultipartForm(10 << 32) // grab the multipart form
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		formdata := r.MultipartForm // ok, no problem so far, read the Form data

		files := formdata.File["files"] // grab the filenames
		for _, f := range files {       // loop through the files one by one
			fmt.Println("fname >", f.Filename)

			file, err := f.Open()
			defer file.Close()
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}

			out, err := os.Create(Keystore + f.Filename)
			defer out.Close()
			if err != nil {
				fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
				return
			}

			_, err = io.Copy(out, file)
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}
		}

	}
}
