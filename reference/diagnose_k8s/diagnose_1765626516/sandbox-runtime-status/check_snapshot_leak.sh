#!/bin/bash

# one snapshot one container
# exit code means leak number

if [ -z $(command -v dm_check) ]; then
    echo "not found dm_check"
    exit 0
fi

config="/etc/container-storaged/config.toml"

if [ ! -f ${config} ]; then
    echo "please install container-storaged, not found config ${config}"
    exit 1
fi

containerdRoot=$(grep containerd_root ${config} | awk -F\" '{print $2}')
if [ -z ${containerdRoot} ]; then
    echo "can not get container_root from ${config}"
    exit 1
fi

dbfile=${containerdRoot}/io.containerd.snapshotter.v1.devmapper/vg0-mythinpool.db
if [ ! -f ${dbfile} ]; then
    echo "can not found dbfile ${dbfile}"
    exit 1
fi

snapshots=`dmsetup ls | grep -c vg0-mythinpool-snap`
containers=`ctr -n k8s.io c ls | grep -vc CONTAINER`
leaknum=$(($snapshots-$containers))

if [ ${leaknum} -gt 0 ]; then
	echo "${leaknum} snapshots leaked"
fi

# exec dm_check
dbdir=/tmp/db/$(date "+%Y%m%d%H%M%S")
mkdir -p ${dbdir}
cp ${dbfile} ${dbdir}
dm_check -db ${dbdir}/vg0-mythinpool.db
rm -fr ${dbdir}

exit ${leaknum}
