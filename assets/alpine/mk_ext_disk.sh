#!/bin/sh

dd if=/dev/zero of=./ext_disk.img bs=100M count=1
mkfs.ext4 -E lazy_itable_init ./ext_disk.img
