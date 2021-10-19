package provisioner

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/mysteriumnetwork/hyperv-node/common"
	"github.com/mysteriumnetwork/hyperv-node/powershell"
)

type Provisioner struct {
	shell       *powershell.PowerShell
	NodeVersion string
}

func NewProvisioner(shell *powershell.PowerShell) (*Provisioner, error) {
	nodeVersion, err := latestNodeVersion()
	if err != nil {
		return nil, err
	}

	return &Provisioner{shell: shell, NodeVersion: nodeVersion}, nil
}

func (p *Provisioner) InstallMystClean(privateKeyPath, user, ip string) error {
	ssh := fmt.Sprintf("ssh -i %s %s@%s", privateKeyPath, user, ip)
	p.exec(ssh, "rm install-myst.sh")
	err := p.exec(ssh, "wget https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/master/assets/alpine/install-myst.sh")
	if err != nil {
		return err
	}

	err = p.exec(ssh, "chmod +x install-myst.sh")
	if err != nil {
		return err
	}

	err = p.exec(ssh, fmt.Sprintf("./install-myst.sh %s", p.NodeVersion))
	if err != nil {
		return err
	}

	err = p.exec(ssh, "rm install-myst.sh")
	if err != nil {
		return err
	}

	return nil
}

func ssh(privateKeyPath, user, ip string) string {
	return fmt.Sprintf("ssh -i %s %s@%s", privateKeyPath, user, ip)
}

func (p *Provisioner) CopyKeystoreRecursive(source, target, privateKeyPath, user, ip string) error {
	ssh := ssh(privateKeyPath, user, ip)
	p.exec(ssh, "mkdir .mysterium")
	return common.OutWithIt(
		p.shell.Execute(
			"scp",
			"-i",
			common.WrapInQuotes(privateKeyPath),
			"-r",
			common.WrapInQuotes(source),
			fmt.Sprintf("root@%s:%s", ip, common.WrapInQuotes(target)),
		))
}

func latestNodeVersion() (string, error) {
	releases, err := gitReleases("mysteriumnetwork", "node", 1)
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", errors.New("did not find a single release of node")
	}
	return releases[0].TagName, nil
}

func (p *Provisioner) exec(ssh, cmd string) error {
	// linux outs to error stream
	_, err := p.shell.Execute(ssh, cmd)
	return err
}
