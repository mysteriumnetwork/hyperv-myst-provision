#!/bin/sh

# Update bootloader settings for VM w/o UEFI support
# echo default_kernel_opts=\\\"rootfstype=ext4 console=ttyS0,115200 cgroup_enable=memory swapaccount=1\\\" >> /etc/update-extlinux.conf
# echo serial_port=0 >> /etc/update-extlinux.conf
# echo serial_baud=115200 >> /etc/update-extlinux.conf
# echo timeout=1 >> /etc/update-extlinux.conf
# update-extlinux

# Add guest services
apk add hvtools
rc-update add hv_fcopy_daemon default
rc-update add hv_kvp_daemon default
rc-update add hv_vss_daemon default
# rc-service hv_fcopy_daemon start
# rc-service hv_kvp_daemon start
# rc-service hv_vss_daemon start

# start crashed services periodically
echo '* * * * * openrc' >> /etc/crontabs/root
rc-update add crond
