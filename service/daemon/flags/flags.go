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

package flags

import (
	"flag"

	"github.com/rs/zerolog"
)

// VM helper CLI flags.
var (
	FlagVersion   = flag.Bool("version", false, "Print version")
	FlagInstall   = flag.Bool("install", false, "Install or repair VM helper")
	FlagUninstall = flag.Bool("uninstall", false, "Uninstall myst VM helper")

	FlagImportVM               = flag.Bool("import", false, "Import myst VM")
	FlagImportVMPreferEthernet = flag.Bool("prefer-ethernet", false, "Prefer Ethernet connection")

	FlagLogFilePath = flag.String("log-path", "", "Log file path")
	FlagLogLevel    = flag.String("log-level", zerolog.InfoLevel.String(), "Logging level")
	FlagWinService  = flag.Bool("winservice", false, "Run via service manager instead of standalone (windows only).")
	FlagVMName      = flag.String("vm-name", "Myst HyperV Alpine", "hyper-v guest VM name")
)

// Parse parses command flags.
func Parse() {
	flag.Parse()
}
