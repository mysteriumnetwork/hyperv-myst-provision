module github.com/mysteriumnetwork/hyperv-node

go 1.16

require (
	github.com/Microsoft/go-winio v0.5.1
	github.com/artdarek/go-unzip v1.0.1-0.20210323073738-f9883ad8bd15
	github.com/blang/semver/v4 v4.0.0
	github.com/dghubble/sling v1.4.0
	github.com/gabriel-samfira/go-wmi v0.0.0-20200311221200-7c023ba1e6b4
	github.com/go-ole/go-ole v1.2.6
	github.com/gonutz/w32 v1.0.0
	github.com/google/glazier v0.0.0-20211213200644-0506347f83ee
	github.com/itzg/go-flagsfiller v1.6.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/mdlayher/vsock v1.1.1
	github.com/mysteriumnetwork/go-fileversion v1.0.0-fix1
	github.com/mysteriumnetwork/myst-launcher v0.0.0-20211221075138-6f8014606bf0
	github.com/mysteriumnetwork/node v0.0.0-20220104164347-5ded05b0ebf0
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.0
	github.com/sevlyar/go-daemon v0.1.5
	github.com/terra-farm/go-virtualbox v0.0.5-0.20220105144057-c7149ec50e96
	github.com/winlabs/gowin32 v0.0.0-20210302152218-c9e40aa88058
	golang.org/x/sys v0.0.0-20220204135822-1c1b9b1eba6a
)

replace (
	//tag:patch-v1
	github.com/gabriel-samfira/go-wmi => github.com/mysteriumnetwork/go-wmi v0.0.0-20211216181752-dbce75057213

	//tag:shlwapi-r4
	github.com/gonutz/w32 => github.com/mysteriumnetwork/w32 v1.0.1-0.20211216070125-4741b8b8111b

	//tag:patch-v1
	github.com/terra-farm/go-virtualbox => github.com/mysteriumnetwork/go-virtualbox v0.0.5-0.20220323071623-231aaecf8949
)
