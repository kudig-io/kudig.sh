#!/bin/bash

# netns is created by cni or kata, if pod not use host network ,it owns one netns
# can not determine actual netns, should not more than containers


container=-1
if rpm -qa | grep -q pouch-container;then
    containers=$(pouch ps -q | wc -l)
else
    containers=$(ps -ef | grep containerd-shim | grep -v grep | wc -l)
fi

mountnetns=$(mount -l | grep "netns\/cni" | wc -l)
state=0

if [[ $mountnetns -gt $containers ]];then
    echo "containers: $containers, mount netns: $mountnetns, netns may leak"
    state=1
fi

exit $state
