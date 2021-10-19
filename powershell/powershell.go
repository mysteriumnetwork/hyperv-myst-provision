package powershell

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Option string

const (
	OptionDebugPrint Option = "OptionDebugPrint"
)

type PowerShell struct {
	powerShell  string
	printOutput bool
}

func New(options ...Option) *PowerShell {
	ps, _ := exec.LookPath("powershell.exe")
	powerShell := &PowerShell{
		powerShell: ps,
	}

	for _, option := range options {
		if option == OptionDebugPrint {
			powerShell.printOutput = true
		}
	}

	return powerShell
}

func (p *PowerShell) Execute(args ...string) (Out, error) {
	args = append([]string{"-NoProfile", "-NonInteractive"}, args...)
	cmd := exec.Command(p.powerShell, args...)

	var out Out
	cmd.Stdout = &out.Out
	cmd.Stderr = &out.Err

	if p.printOutput {
		fmt.Println("Executing:", cmd.String())
	}

	err := cmd.Run()

	if p.printOutput {
		if out.IsErr() {
			fmt.Println(out.ErrString())
		} else {
			if len(out.OutString()) == 0 {
				fmt.Println("Response:", "<empty>")
			} else {
				fmt.Println("Response:", out.OutString())
			}
		}
	}

	return out, err
}

type Out struct {
	Out bytes.Buffer
	Err bytes.Buffer
}

func (o *Out) IsErr() bool {
	return o.Err.Len() > 0
}

func (o *Out) OutString() string {
	return o.Out.String()
}

func (o *Out) OutTrimString(cutset string) string {
	return strings.Trim(o.Out.String(), cutset)
}

func (o *Out) OutTrimNewLineString() string {
	return strings.Trim(o.Out.String(), "\r\n")
}

func (o *Out) IsEmpty() bool {
	return o.OutString() == ""
}

func (o *Out) ErrString() string {
	return o.Err.String()
}

func (o *Out) GetError() error {
	return errors.New(o.ErrString())
}
