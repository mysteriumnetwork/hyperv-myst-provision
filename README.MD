# HyperV image provisioning and distribution

## Fresh Alpine Install

### HyperV Guest daemons
This will allow HyperV guest services to interact and obtain information from guest VM (i.e. IP Address)
```shell
apk add hvtools

rc-update add hv_fcopy_daemon default
rc-update add hv_kvp_daemon default
rc-update add hv_vss_daemon default

rc-service hv_fcopy_daemon start
rc-service hv_kvp_daemon start
rc-service hv_vss_daemon start
```

### Install MYST

Dependencies
```shell
apk add --no-cache iptables ipset ca-certificates openvpn bash sudo openresolv
```

Prepare environment
```shell
wget https://raw.githubusercontent.com/mysteriumnetwork/node/master/bin/helpers/prepare-run-env.sh
chmod +x ./prepare-run-env.sh
./prepare-run-env.sh
```

Download myst node
```shell
wget https://github.com/mysteriumnetwork/node/releases/download/0.66.1/myst_linux_amd64.tar.gz
tar -xvf myst_linux_amd64.tar.gz
rm myst_linux_amd64.tar.gz
cp myst /bin/
```

Copy keystore to `/root/.mysterium/`
```shell
???
```

Register OpenRC service and start it
```shell
wget https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/master/assets/alpine/myst-service
mv myst-service /etc/init.d/
chmod +x /etc/init.d/myst-service
rc-update add myst-service default
rc-service myst-service start
```

All In one:

```shell
mkdir node
cd node
apk add --no-cache iptables ipset ca-certificates openvpn bash sudo openresolv
wget https://raw.githubusercontent.com/mysteriumnetwork/node/master/bin/helpers/prepare-run-env.sh
chmod +x ./prepare-run-env.sh
./prepare-run-env.sh
wget https://github.com/mysteriumnetwork/node/releases/download/0.67.0/myst_linux_amd64.tar.gz
tar -xvf myst_linux_amd64.tar.gz
rm myst_linux_amd64.tar.gz
cp myst /bin/
wget https://raw.githubusercontent.com/mysteriumnetwork/hyperv-myst-provision/master/assets/alpine/myst-service
mv myst-service /etc/init.d/
chmod +x /etc/init.d/myst-service
rc-update add myst-service default
rc-service myst-service start
```