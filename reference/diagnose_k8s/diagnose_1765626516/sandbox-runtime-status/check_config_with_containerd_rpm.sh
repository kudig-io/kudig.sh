#!/bin/bash

# exit status
# 0 -> ok
# 1 -> not ok
# 2 -> unknow

if ! rpm -qa | grep -q containerd;then
	exit 0
fi

graphdriver=$(grep "snapshotter" /etc/containerd/config.toml | awk -F\" '{print $2}')
if [ -z "graphdriver" ];then
	echo "can not get graphdriver from /etc/containerd/config.toml"
fi

# check iommu
if ! $(cat /proc/cmdline | grep -q iommu);then
	echo "please enable iommu"
	exit 1
fi

# check pouch config

containerdConfig="/etc/containerd/config.toml"
kataConfig="/etc/kata-containers/configuration.toml"
containerStorageConfig="/etc/container-storaged/config.toml"

if [ ! -f "$containerdConfig" ] || [ ! -f "$kataConfig" ];then
	echo "please install containerd/kata, config not found"
	exit 1
fi

if [ "$graphdriver" == "devmapper" ];then
	if [ ! -f $containerStorageConfig ]; then
		echo "please install container-storage, config not found"
		exit 1
	fi
	
	if [ -z "$(pidof container-storaged)" ];then
		echo "please start container-storaged service, work for dm"
		exit 1
	fi
	
	if ! $(ctr -a /var/run/containerd/containerd.sock plugin ls | grep devmapper | grep -q ok);then
		echo "please make sure containerd load devmapper successfully"
		exit 1
	fi
fi

if [ -z "$(pidof containerd)" ];then
	echo "please start containerd service"
	exit 1
fi

# check kata config

if ! $(cat "$kataConfig" | grep -q "^disable_9p_file"); then
	echo "please enable disable_9p_file in kata config"
	exit 1
fi

if $(cat "$kataConfig" | grep -q "^enable_hotplugin_res"); then
	echo "please disable enable_hotplugin_res in kata config"
	exit 1
fi

if $(cat "$kataConfig" | grep -q "^disable_default_cpus"); then
	echo "please disable disable_default_cpus in kata config"
	exit 1
fi

# check qemu machine_accelerators parameter

if $(cat "$kataConfig" | grep -q vmlinux); then
	if ! $(cat "$kataConfig" | grep machine_accelerators | grep -q nofw); then
		echo "please add nofw in machine_accelerators in kata config"
	fi
fi

if $(cat "$kataConfig" | grep -q "\/kernel"); then
	if $(cat "$kataConfig" | grep machine_accelerators | grep -q nofw); then
		echo "please remove nofw in machine_accelerators in kata config"
	fi
fi
