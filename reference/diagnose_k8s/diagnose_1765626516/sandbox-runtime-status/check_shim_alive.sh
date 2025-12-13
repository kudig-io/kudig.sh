#!/bin/bash

# include containerd-shim / kata-v2
# consider pouch-container/containerd rpm two case

socket=""
execid=$(date +%s)
hang=0
socketlist=("/var/run/containerd.sock" "/var/run/containerd/containerd.sock")
for s in ${socketlist[@]}; do
	if [ -e "$s" ];then
		socket="$s"
		break
	fi
done

if [ -z "$socket" ];then
	echo "can not find containerd socket"
	exit 10086
fi

# check containerd namespace default
for i in $(ctr -a $socket c ls -q); do
	if [[ $(ps -ef | grep $i | wc -l) -le 1 ]];then
		continue
	fi
	out=$(ctr --timeout 3s -a $socket t exec --exec-id=$execid $i echo 1 2>&1)
	if $(echo "$out" | grep -q "context deadline exceede");then
		hang=$[$hang+1]
		echo "$i stuck"
	fi
done

# check containerd namespace k8s.io
for i in $(ctr -a $socket -n k8s.io c ls -q); do
	if [[ $(ps -ef | grep $i | wc -l) -le 1 ]];then
		continue
	fi
	out=$(ctr --timeout 3s -a $socket t exec --exec-id=$execid $i echo 1 2>&1)
	if $(echo "$out" | grep -q "context deadline exceede");then
		hang=$[$hang+1]
		echo "$i stuck"
	fi
done


exit $hang

