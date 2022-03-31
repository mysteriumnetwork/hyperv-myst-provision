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

package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mysteriumnetwork/hyperv-node/model"
	"github.com/mysteriumnetwork/hyperv-node/service2/daemon/client"
	transport2 "github.com/mysteriumnetwork/hyperv-node/service2/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/vbox"

	"github.com/rs/zerolog/log"
)

// Daemon - vm helper process.
type Daemon struct {
	mgr              *vbox.Manager
	importInProgress bool
	cfg              *model.Config
}

// New creates a new daemon.
func New(manager *vbox.Manager, cfg *model.Config) Daemon {
	d := Daemon{mgr: manager}
	d.cfg = cfg

	return d
}

// Start the daemon. Blocks.
func (d *Daemon) Start(options transport2.Options) error {
	if d.cfg.Enabled {
		d.mgr.StartVM()
	}

	return transport2.Start(d.dialog, options)
}

// dialog talks to the client via established connection.
func (d *Daemon) dialog(conn io.ReadWriter) {
	scan := bufio.NewScanner(conn)
	answer := responder{conn}
	for scan.Scan() {
		line := scan.Bytes()
		log.Debug().Msgf("> %s", line)

		m := make(map[string]interface{}, 0)
		err := json.Unmarshal([]byte(line), &m)
		fmt.Println(m, err)
		op := strings.ToLower(m["cmd"].(string))

		switch op {
		case CommandVersion:
			answer.ok_(nil)

		case CommandPing:
			answer.pong_()

		case CommandStopVM:
			err = d.mgr.StopVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", CommandStopVM)
				answer.err_(err)
			} else {
				answer.ok_(nil)
			}

		case CommandStartVM:
			err := d.mgr.StartVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", CommandStartVM)
				answer.err_(err)
			} else {
				answer.ok_(nil)
			}

		case CommandImportVM:
			reportProgress, _ := m["report-progress"].(bool)
			preferEthernet, _ := m["prefer-ethernet"].(bool)
			keystoreDir, _ := m["keystore"].(string)
			adapterID, _ := m["adapter-id"].(string)
			adapterName, _ := m["adapter-name"].(string)

			if d.importInProgress {
				// prevent parallel runs of import-vm
				answer.err_(errors.New("in progress"))
			} else {
				var fn vbox.ProgressFunc
				if reportProgress {
					fn = func(progress int) {
						answer.progress_(CommandImportVM, progress)
					}
				}
				d.importInProgress = true

				vmInfo := new(vbox.VMInfo)
				err = d.mgr.ImportVM(vbox.ImportOptions{
					Force:                true, //false,
					VMBootPollSeconds:    5,
					VMBootTimeoutMinutes: 1,
					KeystoreDir:          keystoreDir,
					PreferEthernet:       preferEthernet,
					AdapterID:            adapterID,
					AdapterName:          adapterName,
				}, fn, vmInfo)
				if err != nil {
					log.Err(err).Msgf("%s failed >", op)
					answer.err_(err)

				} else {
					log.Info().Msgf("vmInfo> %v %v ", vmInfo.AdapterName, vmInfo.NodeIdentity)
					answer.ok_(vmInfo)

				}
				d.importInProgress = false
			}

		case CommandGetAdapters:
			l, err := d.mgr.SelectAdapter()
			if err != nil {
				log.Err(err).Msgf("%s failed", op)
				answer.err_(err)
			} else {
				answer.ok_(l)
			}

		//case CommandSetAdapter:
		//	adapterID, _ := m["adapter-adapterID"].(string)
		//	d.cfg.AdapterID = adapterID
		//	d.cfg.Save()
		//
		//	l, err := d.mgr.SelectAdapter()
		//	if err != nil {
		//		log.Err(err).Msgf("%s failed", op)
		//		answer.err_(err)
		//	} else {
		//		answer.ok_(l)
		//	}

		case CommandGetVMState:
			var m map[string]interface{}
			m["enabled"] = d.cfg.Enabled
			answer.ok_(m)

		case CommandGetKvp:
			err = d.mgr.GetGuestKVP()
			if err != nil {
				log.Err(err).Msgf("%s failed", op)
				answer.err_(err)
			} else {
				answer.ok_(d.mgr.Kvp)
			}

		case CommandUpdateNode:
			err = d.mgr.GetGuestKVP()
			if err != nil {
				log.Err(err).Msgf("%s failed", op)
				answer.err_(err)
			}
			log.Info().Msgf("%v", d.mgr.Kvp)

			ip, ok := d.mgr.Kvp["NetworkAddressIPv4"].(string)
			if ok && ip != "" {
				log.Info().Msgf("%v", ip)
			}
			client.VmAgentUpdateNode(ip)
			answer.ok_(d.mgr.Kvp)

		}
	}
}
