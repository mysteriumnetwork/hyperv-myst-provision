#!/bin/sh

# Update bootloader settings for VM w/o UEFI support
# echo default_kernel_opts=\\\"rootfstype=ext4 console=ttyS0,115200 cgroup_enable=memory swapaccount=1\\\" >> /etc/update-extlinux.conf
# echo serial_port=0 >> /etc/update-extlinux.conf
# echo serial_baud=115200 >> /etc/update-extlinux.conf
# echo timeout=1 >> /etc/update-extlinux.conf
# update-extlinux

# Add guest services

cat > /etc/apk/repositories << EOF; $(echo)

https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/main/
https://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/community/
https://dl-cdn.alpinelinux.org/alpine/edge/testing/

EOF

apk add virtualbox-guest-additions
rc-update add virtualbox-guest-additions default
rc-update add acpid default

echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config

tee -a /etc/network/interfaces << END

auto eth1
iface eth1 inet dhcp

END


apk add hvtools
rc-update add hv_fcopy_daemon default
rc-update add hv_kvp_daemon default
rc-update add hv_vss_daemon default

#sed -i 's/^ttyS0/#ttyS0/' /etc/inittab

mkdir /mnt/ext
mount /dev/sdb /mnt/ext
cp /mnt/ext/vm-agent /bin/
