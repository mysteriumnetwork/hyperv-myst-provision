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

	"github.com/rs/zerolog/log"

	hyperv_wmi2 "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	"github.com/mysteriumnetwork/hyperv-node/provisioner"
	"github.com/mysteriumnetwork/hyperv-node/service/daemon/model"
	transport2 "github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
)

// Daemon - vm helper process.
type Daemon struct {
	mgr              *hyperv_wmi2.Manager
	importInProgress bool
	cfg              model.Config
}

// New creates a new daemon.
func New(manager *hyperv_wmi2.Manager) Daemon {
	d := Daemon{mgr: manager}
	d.cfg.Read()

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
		case commandVersion:
			answer.ok_(nil)

		case commandPing:
			answer.pong_()

		case commandStopVM:
			err := d.mgr.RemoveSwitch()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStopVM)
				answer.err_(err)
			}
			err = d.mgr.StopVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStopVM)
				answer.err_(err)
			} else {
				d.mgr.RemoveVM()
				answer.ok_(nil)
			}

		case commandStartVM:
			err := d.mgr.StartVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStartVM)
				answer.err_(err)
			} else {
				answer.ok_(nil)
			}

		case commandImportVM:
			reportProgress, _ := m["report-progress"].(bool)
			preferEthernet, _ := m["prefer-ethernet"].(bool)
			keystoreDir, _ := m["keystore"].(string)

			if d.importInProgress {
				// prevent parallel runs of import-vm
				answer.err_(errors.New("in progress"))
			} else {
				var fn provisioner.ProgressFunc
				if reportProgress {
					fn = func(progress int) {
						answer.progress_(commandImportVM, progress)
					}
				}
				d.importInProgress = true
				err = d.mgr.ImportVM(hyperv_wmi2.ImportOptions{
					Force:                true, //false,
					VMBootPollSeconds:    5,
					VMBootTimeoutMinutes: 1,
					KeystoreDir:          keystoreDir,
					PreferEthernet:       preferEthernet,
				}, fn)
				if err != nil {
					log.Err(err).Msgf("%s failed", commandImportVM)
					answer.err_(err)
				} else {
					answer.ok_(nil)
				}
				d.importInProgress = false
			}

		case commandGetVMState:
			var m map[string]interface{}
			m["enabled"] = d.cfg.Enabled
			answer.ok_(m)

		case commandGetKvp:
			err = d.mgr.GetGuestKVP()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandGetKvp)
				answer.err_(err)
			} else {
				answer.ok_(d.mgr.Kvp)
			}

		}
	}
}
