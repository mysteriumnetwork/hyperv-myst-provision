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
	"github.com/Microsoft/go-winio"
	"net"
)

const sock = `\\.\pipe\myst-vm-helper-pipe`

func sendCommand(conn net.Conn, m map[string]string) {
	b, _ := json.Marshal(m)
	fmt.Println("send > " + string(b))
	conn.Write(b)
	conn.Write([]byte("\n"))

	out := make([]byte, 100)
	conn.Read(out)
	fmt.Println("rcv >", string(out))
}

func main() {
	conn, err := winio.DialPipe(sock, nil)
	if err != nil {
		fmt.Printf("error listening: %v", err)
		return
	}
	defer conn.Close()

	m := map[string]string{"cmd": "ping"}
	sendCommand(conn, m)

	m = map[string]string{
		"cmd":      "import-vm",
		"keystore": `C:\Users\user\.mysterium\keystore`,
		"work-dir": `C:\Users\user\src\work_dir\alpine-vm-disk\`}
	sendCommand(conn, m)

}
