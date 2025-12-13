#!/bin/bash

# alarm if current fd almost get max value

totalFD=$(cat /proc/sys/fs/file-max)
curFD=$(cat /proc/sys/fs/file-nr|awk '{print $1}')

if [[ $[totalFD-curFD] -lt 1000 ]];then
    echo "fd may exceed"
fi

# try get all open fd from /proc
curFD=$(ls /proc | while read x; do
    if [[ ! -d "/proc/$x" ]] || [[ -n "`echo "$x" | grep [a-z\|A-Z]`" ]];then
        continue
    fi
    find /proc/${x% *}/task/${x#* }/fd/ -type l;
done | wc -l)

if [[ $[totalFD-curFD] -lt 1000 ]];then
    echo "fd may exceed"
fi


exit 0
