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
	"github.com/mysteriumnetwork/hyperv-node/daemon/transport"
	"github.com/mysteriumnetwork/hyperv-node/hyperv/network"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
)

// Daemon - vm helper process.
type Daemon struct {
	mgr *network.Manager
}

// New creates a new daemon.
func New(manager *network.Manager) Daemon {
	return Daemon{mgr: manager}
}

// Start supervisor daemon. Blocks.
func (d *Daemon) Start(options transport.Options) error {
	return transport.Start(d.dialog, options)
}

// dialog talks to the client via established connection.
func (d *Daemon) dialog(conn io.ReadWriter) {
	scan := bufio.NewScanner(conn)
	answer := responder{conn}
	for scan.Scan() {
		line := scan.Bytes()
		log.Debug().Msgf("> %s", line)
		cmd := strings.Split(string(line), " ")
		op := strings.ToLower(cmd[0])
		switch op {
		case commandVersion:
			answer.ok("")

		case commandPing:
			answer.ok("pong")

		case commandStopVM:
			err := d.mgr.StopVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStopVM)
				answer.err(err)
			} else {
				answer.ok()
			}

		case commandStartVM:
			err := d.mgr.StartVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStartVM)
				answer.err(err)
			} else {
				answer.ok()
			}
		}
	}
}
