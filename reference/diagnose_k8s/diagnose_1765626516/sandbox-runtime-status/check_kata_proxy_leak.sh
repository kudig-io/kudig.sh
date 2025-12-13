#!/bin/bash

# one pod one kata-proxy
# only take effect in kata shim v1

# in ack case, do not need this check

if ! rpm -qa | grep -q pouch-container;then
	exit 0
fi

if [[ "$(pouch ps 2>/dev/null | wc -l)" == "0" ]];then
	echo "pouch not running"
	exit 10086
fi

if $(cat /etc/pouch/config.json | grep -q "io.containerd.kata.v2");then
	exit 0
fi

alived=$(pouch ps -a --no-trunc | grep kata | grep pause-amd | awk '{print $2}' | xargs)
leakid=""
leaknum=0

for i in $(pidof kata-proxy);do
	pid="$i"
	id=$(cat /proc/$pid/cmdline | awk -F -sandbox '{print $NF}')

	if [[ $alived == *$id* ]]; then 
		continue
	fi

	echo "leak kata proxy $id"
	leaknum=$[$leaknum+1]

done

exit $leaknum

