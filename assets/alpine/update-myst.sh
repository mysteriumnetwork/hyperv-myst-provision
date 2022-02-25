#!/bin/sh

arch=$1

cd /root
mkdir node
cd node
apk add --no-cache iptables ipset ca-certificates bash sudo openresolv

wget https://github.com/mysteriumnetwork/node/releases/latest/download/myst_linux_${arch}.tar.gz
tar -xvf myst_linux_amd64.tar.gz
rm myst_linux_amd64.tar.gz
cp myst /bin/

wget https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/master/assets/alpine/myst-service
mv myst-service /etc/init.d/
chmod +x /etc/init.d/myst-service
rc-update add myst-service default

cd ..
rm -rf node