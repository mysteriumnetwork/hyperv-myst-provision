#!/bin/sh

setup-keymap us us
setup-hostname myst-node

# Setup networking
cat << END > /etc/network/interfaces
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp
    hostname myst-node
END

rc-service networking start

setup-timezone -z UTC

echo -e "$ROOT_PASSWORD\n$ROOT_PASSWORD" | passwd

setup-sshd -c openssh

# Enable root login over SSH
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak
sed 's/#PermitRootLogin.*$/PermitRootLogin yes/' /etc/ssh/sshd_config > /etc/ssh/sshd_config
rc-service sshd restart

# Random setup of repos
setup-apkrepos -r

setup-ntp -c busybox

# Setup time sync
setup-ntp -c busybox

# Partition disk and install alpine linux
echo y| setup-disk -s0 -m sys /dev/sda

#poweroff