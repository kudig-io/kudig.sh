#!/bin/bash

# one pod one qemu
# exit code means leak number


alived=""
if [ -n "`pidof pouchd`" ];then
    alived=$(pouch ps --no-trunc | grep kata | awk '{print $2}' | xargs)
fi

if [ -n "`command -v ctr`" ];then
    # consider if containerd-shim-kata-v2 alive, qemu is not leak
    alived=$(ps -ef | grep containerd-shim-kata-v2 | grep -v grep | awk '{print $(NF-1)}'| xargs)
fi

leakid=""
leaknum=0

for i in $(ps -ef | grep qemu-kvm | grep -v "grep" | awk '{print$2$10}');do
    qpid=$(echo "$i" | awk -F "sandbox-" '{print $1}')
    id=$(echo "$i" | awk -F "sandbox-" '{print $2}')

    if [ -z "$id" ];then
        echo "unexpected qemu process $i"
        continue
    fi

    if [[ $alived == *$id* ]]; then 
        continue
    fi


    echo "leak qemu-id $id pid $qpid"
    leaknum=$[$leaknum+1]

done

exit $leaknum

