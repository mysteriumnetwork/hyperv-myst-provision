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

	state int
}

// New creates a new daemon.
func New(manager *vbox.Manager, cfg *model.Config) Daemon {
	d := Daemon{mgr: manager}
	d.cfg = cfg

	return d
}

// Start the daemon. Blocks.
func (d *Daemon) Start(options transport2.Options) error {
	log.Info().Msgf("Daemon !Start > %v", options)

	if d.cfg.Enabled {
		d.state = 1
		// 	err := d.mgr.StartVM()
		// 	log.Info().Msgf("Daemon !StartVM %v", err)
	}

	networkChangeChn := make(chan bool)
	go d.mgr.MonitorNetwork(networkChangeChn)
	go func() {
		for {
			select {
			case <-networkChangeChn:
				log.Info().Msgf("> networkChangeEv >>>> %v", d.mgr.MinAdapter)

				// network is online: re-start VM (if VM is enabled)

				if d.state == 1 {
					err := d.mgr.SetNicAndRestartVM()
					if err != nil {
						log.Err(err).Msgf("SetNicAndRestartVM failed")
					}
				}

			}
		}
	}()

	return transport2.Start(d.dialog, options)
}

// dialog talks to the client via established connection.
func (d *Daemon) dialog(conn io.ReadWriteCloser) {
	log.Info().Msg("Daemon !dialog >>>")

	answer := responder{conn}
	lines := make(chan interface{})

	go func() {
		scan := bufio.NewScanner(conn)
		for scan.Scan() {
			b := scan.Bytes()
			lines <- b
		}
		lines <- scan.Err()
	}()

	log.Info().Msg("Daemon !dialog")

	log.Info().Msgf("> Wait for network is ready")
	//select {
	//case <-networkChangeChn:
	//	log.Info().Msgf("> networkChangeEv >>>> %v", d.mgr.MinAdapter)
	//case <-time.After(3 * time.Second):
	//	fmt.Println("networkChangeEv timeout")
	//}

	for {
		select {
		case l := <-lines:

			switch line := l.(type) {
			case []byte:
				// log.Debug().Msgf("> %s", line)
				log.Info().Msgf("Daemon !dialog: %v", string(line))

				m := make(map[string]interface{})
				_ = json.Unmarshal([]byte(line), &m)
				// fmt.Println(m, err)

				op := strings.ToLower(m["cmd"].(string))
				log.Info().Msg("Daemon !dialog !doOperation")
				d.doOperation(op, answer, m)
				log.Info().Msg("Daemon !dialog !doOperation")

			case error:
				// here v has type S

			default:
				// no match; here v has the same type as i
			}

		}
	}
}

func (d *Daemon) doOperation(op string, answer responder, m map[string]interface{}) {

	switch op {
	case CommandVersion:
		answer.ok_(nil)

	case CommandPing:
		answer.pong_()

	case CommandStopVM:
		d.state = 0
		err := d.mgr.StopVM()
		if err != nil {
			log.Err(err).Msgf("%s failed", CommandStopVM)
			answer.err_(err)
		} else {
			answer.ok_(nil)
		}

	case CommandStartVM:
		d.state = 1

		// start VM only if network is online
		if !d.mgr.IsNetworkOnline() {
			err := errors.New("Network is not online")
			log.Err(err).Msgf("%s failed", CommandStartVM)
			answer.err_(err)
		}

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
			err := d.mgr.ImportVM(vbox.ImportOptions{
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
		m := make(map[string]interface{})
		m["enabled"] = d.cfg.Enabled
		answer.ok_(m)

	case CommandGetKvp:
		err := d.mgr.GetGuestKVP()
		if err != nil {
			log.Err(err).Msgf("%s failed", op)
			answer.err_(err)
		} else {
			answer.ok_(d.mgr.Kvp)
		}

	case CommandUpdateNode:
		err := d.mgr.GetGuestKVP()
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
