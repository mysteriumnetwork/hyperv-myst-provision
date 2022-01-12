package util

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/mysteriumnetwork/myst-launcher/native"
)

func PanicHandler(threadName string) {
	if err := recover(); err != nil {

		dir, _ := os.Getwd()
		fmt.Printf("Stacktrace %s: %s\n", threadName, debug.Stack())
		fname := fmt.Sprintf("%s/launcher_trace_%d.txt", dir, time.Now().Second())
		f, err := os.Create(fname)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()

		var bu bytes.Buffer

		bu.WriteString(fmt.Sprintf("Stacktrace %s: \n", threadName))
		bu.Write(debug.Stack())
		f.Write(bu.Bytes())
	}
}

func OpenUrlInBrowser(url string) {
	native.ShellExecuteAndWait(
		0,
		"",
		"rundll32",
		"url.dll,FileProtocolHandler "+url,
		"",
		syscall.SW_NORMAL)
}
