#!/bin/sh

NODE_VERSION=$1

if [ -z "$NODE_VERSION" ]; then
        echo " > Missing NODE VERSION (i.e. 0.67.0)"
        exit 1
fi

mkdir node
cd node
apk add --no-cache iptables ipset ca-certificates bash sudo openresolv wireguard-tools
wget https://raw.githubusercontent.com/mysteriumnetwork/node/master/bin/helpers/prepare-run-env.sh
chmod +x ./prepare-run-env.sh
./prepare-run-env.sh

#wget https://github.com/mysteriumnetwork/node/releases/download/${NODE_VERSION}/myst_linux_amd64.tar.gz
wget https://github.com/mysteriumnetwork/node/releases/download/1.4.11/myst_linux_${arch}.tar.gz

tar -xvf myst_linux_amd64.tar.gz
rm myst_linux_amd64.tar.gz
cp myst /bin/

#wget https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/master/assets/alpine/myst-service
#mv myst-service /etc/init.d/
#chmod +x /etc/init.d/myst-service
#rc-update add myst-service default
cd ..
rm -rf node