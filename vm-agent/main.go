package main

import (
	//"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"

	"github.com/mysteriumnetwork/hyperv-node/vm-agent/utils"
	//daemon "github.com/sevlyar/go-daemon"
	"github.com/mdlayher/vsock"
	"github.com/oklog/run"
)

// To terminate the daemon use:
//  kill `cat sample.pid`
func main() {
	//cntxt := &daemon.Context{
	//	PidFileName: "sample.pid",
	//	PidFilePerm: 0644,
	//	LogFileName: "sample.log",
	//	LogFilePerm: 0640,
	//	WorkDir:     "./",
	//	Umask:       027,
	//	Args:        []string{"[vm-myst-agent]"},
	//}
	//
	//d, err := cntxt.Reborn()
	//if err != nil {
	//	log.Fatal("Unable to run: ", err)
	//}
	//if d != nil {
	//	return
	//}
	//defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("daemon started")

	log.Print(os.Args)
	serve()
	log.Println("daemon exit")
}

func serve() {
	//http.HandleFunc("/", httpHandler)
	//http.ListenAndServe("127.0.0.1:8080", nil)

	pr := utils.NewProcessRunner("sleep", "5")
	l, err := vsock.Listen(30, nil)
	log.Println(l, err)

	// Shutdown gracefully
	{
		var g run.Group
		s := utils.NewSigTermHandler()

		g.Add(func() error { return s.Wait() }, func(err error) { s.Stop() })
		g.Add(func() error { return pr.Start() }, func(err error) { pr.Shutdown() })
		g.Run()
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request from %s: %s %q", r.RemoteAddr, r.Method, r.URL)
	fmt.Fprintf(w, "go-daemon: %q", html.EscapeString(r.URL.Path))
}
