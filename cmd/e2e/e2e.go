/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"

	consts "github.com/mysteriumnetwork/hyperv-node/const"
)

func sendCommand(conn net.Conn, m map[string]interface{}) {
	b, _ := json.Marshal(m)
	fmt.Println("send > " + string(b))
	conn.Write(b)
	conn.Write([]byte("\n"))

	out := make([]byte, 2000)

	// wait for final response
	for {
		n, _ := conn.Read(out)
		if n > 0 {
			var res map[string]interface{}
			payload := out[:n-1]
			fmt.Println("rcv >>", string(payload))

			json.Unmarshal(payload, &res)
			if res["resp"] == "error" || res["resp"] == "ok" || res["resp"] == "pong" {
				break
			}
			if res["resp"] == "progress" {
				fmt.Println("Progress >", res["progress"])
			}
		}
	}

}

func main() {
	conn, err := winio.DialPipe(consts.Sock, nil)
	if err != nil {
		fmt.Printf("error listening: %v", err)
		return
	}
	defer conn.Close()

	cmd := map[string]interface{}{"cmd": "ping"}
	sendCommand(conn, cmd)

	cmd = map[string]interface{}{
		"cmd":             "import-vm",
		"keystore":        `C:\Users\user\.mysterium\keystore`,
		"report-progress": true,
	}
	sendCommand(conn, cmd)

	cmd = map[string]interface{}{
		"cmd": "get-kvp",
	}
	sendCommand(conn, cmd)
}
