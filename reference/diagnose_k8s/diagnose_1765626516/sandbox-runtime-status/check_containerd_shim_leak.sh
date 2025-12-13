#!/bin/bash

# for containerd shim v1, one kata container one shim
# for containerd shim v2, one kata pod one shim v2

leakid=""
leaknum=0

export leaknum

function check_with_pouch()
{
    # check shim v2
    if $(cat /etc/pouch/config.json | grep -q "io.containerd.kata.v2");then
        alived=$(pouch ps --no-trunc | grep kata | awk '{print $2}' | xargs)
        while read line;do
            pid=$(echo "$line" | awk '{print $1}')
            id=$(echo "$line" | awk '{print $10}')
            if [[ $alived == *$id* ]]; then
                continue
            fi

            echo "leak kata shim-v2 pid $pid id $id"
            leaknum=$[$leaknum+1]
        done <<< "$(ps -eo pid,cmd | grep containerd-shim-kata-v2 | grep -v grep)"
    else
        alived=$(pouch ps -aq --no-trunc)
        while read line;do
            pid=$(echo "$line" | awk '{print $1}')
            id=$(echo "$line" | awk '{print $6}' | awk -F\/ '{print $NF}')

            if [[ $alived == *$id* ]]; then
                continue
            fi

            echo "leak shim-v1 pid $pid id $id"
            leaknum=$[$leaknum+1]
        done <<< "$(ps -eo pid,cmd | grep containerd-shim | grep -v grep)"
    fi
}

function check_with_containerd()
{
    alived=$(ctr -a /var/run/containerd/containerd.sock -n k8s.io c ls -q)
    while read line;do
        if $(echo $line | grep -v grep | grep containerd-shim-kata-v2 -q); then
            pid=$(echo "$line" | awk '{print $1}')
            id=$(echo "$line" | awk '{print $10}')
        else
            pid=$(echo "$line" | awk '{print $1}')
            id=$(echo "$line" | awk '{print $6}' | awk -F\/ '{print $NF}')
        fi
        if [[ $alived == *$id* ]]; then
            continue
        fi

        echo "leak shim-v1 pid $pid id $id"
        leaknum=$[$leaknum+1]
    done <<< "$(ps -eo pid,cmd | grep containerd-shim-kata-v2 | grep -v grep)"
}

if rpm -qa | grep -q pouch-container;then
    check_with_pouch
elif rpm -qa | grep -q containerd;then
    check_with_containerd
fi
exit $leaknum
