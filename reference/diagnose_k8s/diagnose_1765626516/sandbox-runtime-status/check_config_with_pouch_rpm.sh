#!/bin/bash

# exit status
# 0 -> ok
# 1 -> not ok
# 2 -> unknow

if ! rpm -qa | grep -q pouch-container;then
	exit 0
fi

graphdriver=$(grep "\"snapshotter\"" /etc/pouch/config.json | awk -F\" '{print $4}')
if [ -z "graphdriver" ];then
	echo "can not get graphdriver from /etc/pouch/config.json"
fi

# check iommu
if ! $(cat /proc/cmdline | grep -q iommu);then
	echo "please enable iommu"
	exit 1
fi


if [ "$graphdriver" != "devmapper" ];then
	# check quota is on
	graph=$(grep "home-dir" /etc/pouch/config.json | awk -F\" '{print $4}')
	mountPath=$(df $graph | tail -1 | awk '{print $NF}')
	if [ "$mountPath" == "" ];then
		echo "can not find graph dir in /etc/pouch/config.json"
		exit 1
	fi
	# for 4.9 and 3.10
	if $(uname -r | grep -q 4.9);then
		if ! $(mount -l | grep "$graph" | grep -q prjquota);then
			echo "please enable prjquota on device $graph mount"
			exit 1
		fi
	fi
	if $(uname -r | grep -q 3.10);then
		if ! $(mount -l | grep "$graph" | grep -q grpquota);then
			echo "please enable grpquota on device $graph mount"
			exit 1
		fi
	fi
fi

# check pouch config

pouchConfig="/etc/pouch/config.json"
kataConfig="/etc/kata-containers/configuration.toml"

if [ ! -f "$pouchConfig" ] || [ ! -f "$kataConfig" ];then
	echo "please install pouch/kata, config not found"
	exit 1
fi

if [ "$graphdriver" == "devmapper" ];then
	if ! $(cat "$pouchConfig" | grep -q devmapper ); then
		echo "please use devmapper in pouch config"
		exit 1
	fi
	
	if [ -z "$(pidof container-storaged)" ];then
		echo "please start container-storaged service, work for dm"
		exit 1
	fi
	
	if ! $(ctr -a /run/containerd.sock plugin ls | grep devmapper | grep -q ok);then
		echo "please make sure containerd load devmapper successfully"
		exit 1
	fi
fi

# check pouch/eni_server is running
if [ -z "$(pidof eni_server)" ];then
	echo "please start eni_server service, work for eni"
	exit 1
fi

if [ -z "$(pidof pouchd)" ];then
	echo "please start pouch service"
	exit 1
fi

# check kata config

if  $(cat "$kataConfig" | grep -q "^disable_9p_file"); then
	echo "please disable disable_9p_file in kata config"
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
