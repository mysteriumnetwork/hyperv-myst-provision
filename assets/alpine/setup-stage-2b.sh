#!/bin/sh



cat > /etc/apk/repositories << EOF; $(echo)

https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/main/
https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/community/
https://dl-cdn.alpinelinux.org/alpine/edge/testing/

EOF

apk update
apk add go git

cd /root/
git clone https://github.com/mysteriumnetwork/hyperv-myst-provision/
cd hyperv-myst-provision
git switch mvp/vbox
go build -ldflags "-s -w" -o bin/vm-agent vm-agent/main.go
ls -la bin/

mkdir /mnt/ext
mount /dev/sdb /mnt/ext
cp bin/vm-agent /mnt/ext
