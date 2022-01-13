module github.com/mysteriumnetwork/hyperv-node

go 1.16

require (
	github.com/Microsoft/go-winio v0.5.1
	github.com/artdarek/go-unzip v1.0.1-0.20210323073738-f9883ad8bd15
	github.com/dghubble/sling v1.4.0
	github.com/gabriel-samfira/go-wmi v0.0.0-20200311221200-7c023ba1e6b4
	github.com/go-ole/go-ole v1.2.6
	github.com/gonutz/w32 v1.0.0 // indirect
	github.com/itzg/go-flagsfiller v1.6.0
	github.com/mysteriumnetwork/myst-launcher v0.0.0-20211221075138-6f8014606bf0
	github.com/mysteriumnetwork/node v0.0.0-20220104164347-5ded05b0ebf0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.0
	golang.org/x/sys v0.0.0-20211213223007-03aa0b5f6827
)

//tag:patch-v1
//github.com/gabriel-samfira/go-wmi => github.com/mysteriumnetwork/go-wmi v0.0.0-20211216181752-dbce75057213

//tag:shlwapi-r4
replace github.com/gonutz/w32 => github.com/mysteriumnetwork/w32 v1.0.1-0.20211216070125-4741b8b8111b
