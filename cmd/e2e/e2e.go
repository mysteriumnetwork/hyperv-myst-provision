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
	"path"

	"golang.org/x/sys/windows"

	consts "github.com/mysteriumnetwork/hyperv-node/const"
	hyperv_wmi "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/client"

	"github.com/Microsoft/go-winio"
)

func main() {
	homeDir, err := windows.KnownFolderPath(windows.FOLDERID_Profile, windows.KF_FLAG_CREATE)
	if err != nil {
		fmt.Printf("error getting profile path: %v", err)
		return
	}
	keystorePath := path.Join(homeDir, consts.KeystorePath)

	conn, err := winio.DialPipe(consts.Sock, nil)
	if err != nil {
		fmt.Printf("error listening: %v", err)
		return
	}
	defer conn.Close()

	cmd := hyperv_wmi.KVMap{"cmd": "ping"}
	client.SendCommand(conn, cmd)

	cmd = hyperv_wmi.KVMap{
		"cmd":             "import-vm",
		"keystore":        keystorePath,
		"report-progress": true,
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
	fmt.Println("KV", kv)
}
