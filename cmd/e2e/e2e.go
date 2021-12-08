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
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

const sock = `\\.\pipe\myst-vm-helper-pipe`

func sendCommand(conn net.Conn, cmd string) {
	fmt.Println("send > " + cmd)
	conn.Write([]byte(cmd + "\n"))

	b := make([]byte, 100)
	conn.Read(b)
	fmt.Println("rcv >", string(b))
}

func main() {
	conn, err := winio.DialPipe(sock, nil)
	if err != nil {
		fmt.Printf("error listening: %v", err)
		return
	}
	defer conn.Close()

	sendCommand(conn, "ping")
	sendCommand(conn, "start-vm")
	time.Sleep(10 * time.Second)
	sendCommand(conn, "stop-vm")
}
