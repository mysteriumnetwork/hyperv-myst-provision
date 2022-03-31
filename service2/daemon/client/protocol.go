package client

import (
	"encoding/json"
	"fmt"
	//"log"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/mysteriumnetwork/hyperv-node/vbox"
)

func SendCommand(conn net.Conn, m vbox.KVMap) vbox.KVMap {
	b, _ := json.Marshal(m)
	log.Debug().Msgf("send > %v", string(b))
	conn.Write(b)
	conn.Write([]byte("\n"))

	out := make([]byte, 2000)

	// wait for final response
	printProgress := false
	for {
		n, _ := conn.Read(out)
		if n > 0 {
			var res map[string]interface{}
			payload := out[:n-1]
			log.Debug().Msgf("rcv < %v", string(payload))

			json.Unmarshal(payload, &res)
			if res["resp"] == "error" || res["resp"] == "pong" {
				return res
			} else if res["resp"] == "ok" {
				return res
			} else if res["resp"] == "progress" {
				fmt.Printf("\rDownload progress: %v%%", res["progress"])
			}
		}
	}
	if printProgress {
		fmt.Println("")
	}

	return nil
}
