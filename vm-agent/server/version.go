package server

import (
	"bufio"
	"bytes"
	"regexp"

	"github.com/mysteriumnetwork/hyperv-node/vm-agent/utils"
)

var versionRegex = regexp.MustCompile(`(?i)\s*version: \s*([^\s]*)\s*`)

//
//returns a node version
// like "1.1.9"
//

func GetNodeVersion() string {
	out := bytes.Buffer{}
	utils.CmdRun(&out, "/bin/myst", "-v")
	r := bytes.NewReader(out.Bytes())
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		match := versionRegex.MatchString(line)
		if !match {
			continue
		}
		parts := versionRegex.FindAllStringSubmatch(line, -1)
		version := parts[0][1]
		return version
	}
	return ""
}
