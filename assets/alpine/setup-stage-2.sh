#!/bin/sh

# Update bootloader settings for VM w/o UEFI support
# echo default_kernel_opts=\\\"rootfstype=ext4 console=ttyS0,115200 cgroup_enable=memory swapaccount=1\\\" >> /etc/update-extlinux.conf
# echo serial_port=0 >> /etc/update-extlinux.conf
# echo serial_baud=115200 >> /etc/update-extlinux.conf
# echo timeout=1 >> /etc/update-extlinux.conf
# update-extlinux

apk add hvtools

rc-update add hv_fcopy_daemon default
rc-update add hv_kvp_daemon default
rc-update add hv_vss_daemon default
# rc-service hv_fcopy_daemon start
# rc-service hv_kvp_daemon start
# rc-service hv_vss_daemon start

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
#rc-service myst-service start