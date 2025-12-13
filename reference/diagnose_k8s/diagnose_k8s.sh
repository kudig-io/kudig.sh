#!/usr/bin/env bash

set -x

#check sudo
if [ "$(whoami)" != "root" ]; then
    echo "User $current_user has no permission to execute this script!"
    exit 1
fi

current_dir=$(pwd)
tmpdir=/tmp
timestamp=$(date +%s)
diagnose_dir=/tmp/diagnose_${timestamp}
mkdir -p $diagnose_dir
daemon_status_dir=${diagnose_dir}/daemon_status
mkdir -p $daemon_status_dir
is_ps_hang=false

run() {
    echo
    echo "-----------------run $@------------------"
    timeout 10s $@
    if [ "$?" != "0" ]; then
        echo "failed to collect info: $@"
    fi
    echo "------------End of ${1}----------------"
}

# check os type
os_env() {
    grep -q "Red Hat" /etc/redhat-release && export OS="RedHat" && return
    grep -q "CentOS Linux" /etc/os-release && export OS="CentOS" && return
    grep -q "Aliyun Linux" /etc/os-release && export OS="AliyunOS" && return
    grep -q "Alibaba Cloud Linux Lifsea" /etc/os-release && export OS="ContainerOS" && return
    grep -q "Alibaba Cloud Linux" /etc/os-release && export OS="AliyunOS" && return
    grep -q "Alibaba Group Enterprise Linux" /etc/os-release && export OS="AliOS" && return
    grep -q "Kylin Linux Advanced Server V10" /etc/os-release && export OS="KylinV10" && return
    grep -q "UnionTech OS Server" /etc/os-release && export OS="CentOS" && return
    grep -q "Anolis OS" /etc/os-release && export OS="CentOS" && return

    echo "ERROR: unknown os... exit."
    return 1
}

dist() {
    cat /etc/issue*
}

command_exists() {
    command -v "$@" > /dev/null 2>&1
}

# Service status
service_status() {
    run service ntpd status | tee -a $diagnose_dir/service_status   
    run service chronyd status | tee -a $diagnose_dir/service_status
    run service kubelet status | tee -a $diagnose_dir/service_status
    run service edge-hub status | tee -a $diagnose_dir/service_status
    run service firewalld status | tee -a $diagnose_dir/service_status

    if command -v pouch > /dev/null 2>&1; then
        run service pouch status | tee -a $diagnose_dir/service_status
    else
    run service docker status | tee -a $diagnose_dir/service_status
    run service containerd status | tee -a $diagnose_dir/service_status
  fi
}


#system info

system_info() {
    run echo ${OS} | tee -a ${diagnose_dir}/system_info
    run uname -a | tee -a ${diagnose_dir}/system_info
    run uname -r | tee -a ${diagnose_dir}/system_info
    run dist | tee -a ${diagnose_dir}/system_info
    if command_exists lsb_release; then
        run lsb_release | tee -a ${diagnose_dir}/system_info
    fi
    run cat /etc/os-release | tee -a ${diagnose_dir}/system_info
    run cat /etc/redhat-release | tee -a ${diagnose_dir}/system_info
    run ulimit -a | tee -a ${diagnose_dir}/system_info
    run sysctl -a | tee -a ${diagnose_dir}/system_info
    run cat /proc/vmstat | tee -a ${diagnose_dir}/system_info
}

#network
network_info() {
    run ip --details ad show | tee -a ${diagnose_dir}/network_info
    run ip --details link show | tee -a ${diagnose_dir}/network_info
    run ip route show | tee -a ${diagnose_dir}/network_info
    run iptables-save | tee -a ${diagnose_dir}/network_info
    run cat /proc/net/nf_conntrack | tee -a ${diagnose_dir}/network_info
    netstat -nt | tee -a ${diagnose_dir}/network_info
    netstat -nu | tee -a ${diagnose_dir}/network_info
    netstat -ln | tee -a ${diagnose_dir}/network_info
}

memory_info() {
    run cat /proc/meminfo | tee -a ${diagnose_dir}/memory_info
    run cat /proc/buddyinfo | tee -a ${diagnose_dir}/memory_info
    run cat /proc/vmallocinfo | tee -a ${diagnose_dir}/memory_info
    run cat /proc/slabinfo | tee -a ${diagnose_dir}/memory_info
    run cat /proc/zoneinfo | tee -a ${diagnose_dir}/memory_info
}


# check ps -ef command is hung
check_ps_hang() {
  echo "check if ps -ef command hang" | tee -a ${diagnose_dir}/ps_command_status
  checkD=$(timeout -s 9 2 ps -ef)
  if [ "$?" != "0" ]; then
      echo "ps -ef command is hung" | tee -a ${diagnose_dir}/ps_command_status
      is_ps_hang=true
      echo "start to check which process lead to ps -ef command hang" | tee -a ${diagnose_dir}/ps_command_status
      for f in `find /proc/*/task -name status`
      do
          checkD=$(cat $f|grep "State.*D")
          if [ "$?" == "0" ]; then
              cmdline=$(echo ${f%%status}"cmdline")
              pid=$(echo ${f%%status}"")
              stack=$(echo ${f%%status}"stack")
              timeout -s 9 2 cat $cmdline
              if [ "$?" != "0" ]; then
                  echo "process $pid is in State D and lead to ps -ef process hang,stack info:" | tee -a ${diagnose_dir}/ps_command_status
                  cat $stack | tee -a ${diagnose_dir}/ps_command_status
              fi
          fi
      done
      echo "finish to check which process lead to ps -ef command hang" | tee -a ${diagnose_dir}/ps_command_status
  else
      echo "ps -ef command works fine" | tee -a ${diagnose_dir}/ps_command_status
  fi
}


#system status
system_status() {
    #mkdir -p ${diagnose_dir}/system_status
    run uptime | tee -a ${diagnose_dir}/system_status
    run top -b -n 1 | tee -a ${diagnose_dir}/system_status
    if [ "$is_ps_hang" == "false" ]; then
        run ps -ef | tee -a ${diagnose_dir}/system_status
    else
        echo "ps -ef command hang, skip [ps -ef] check" | tee -a ${diagnose_dir}/system_status
    fi
    run netstat -nt | tee -a ${diagnose_dir}/system_status
    run netstat -nu | tee -a ${diagnose_dir}/system_status
    run netstat -ln | tee -a ${diagnose_dir}/system_status

    run sar -A | tee -a ${diagnose_dir}/system_status

    run df -h | tee -a ${diagnose_dir}/system_status

    run cat /proc/mounts | tee -a ${diagnose_dir}/system_status

    run rpm -qa | tee -a ${diagnose_dir}/system_status

    run dbus-send --system --dest=org.freedesktop.systemd1 --type=method_call --reply-timeout=1 --print-reply /org/freedesktop/systemd1  org.freedesktop.DBus.Introspectable.Introspect | tee -a ${diagnose_dir}/system_status

    if [ "$is_ps_hang" == "false" ]; then
        run pstree -al | tee -a ${diagnose_dir}/system_status
    else
        echo "ps -ef command hang, skip [pstree -al] check" | tee -a ${diagnose_dir}/system_status
    fi

    runc_task_status

    run lsof | tee -a ${diagnose_dir}/system_status

    (
        cd /proc
        find -maxdepth 1 -type d -name '[0-9]*' \
         -exec bash -c "ls {}/fd/ | wc -l | tr '\n' ' '" \; \
         -printf "fds (PID = %P), command: " \
         -exec bash -c "tr '\0' ' ' < {}/cmdline" \; \
         -exec echo \; | sort -rn | head | tee -a ${diagnose_dir}/system_status
    )

    echo "----------------start pid leak detect---------------------" | tee -a ${diagnose_dir}/system_status
    ps -elT | awk '{print $4}' | sort | uniq -c | sort -k 1 -g | tail -5 | tee -a ${diagnose_dir}/system_status
    echo "----------------done pid leak detect---------------------" | tee -a ${diagnose_dir}/system_status
}

runc_task_status() {
    for p_runc in `pidof runc`; do
        run ps -p $p_runc -u | tee -a ${diagnose_dir}/system_status
        echo "runc process $p_runc maybe hang, start print stack info:" | tee -a ${diagnose_dir}/system_status
        cat /proc/$p_runc/stack | tee -a ${diagnose_dir}/system_status
        cat /proc/$p_runc/task/*/stack | tee -a ${diagnose_dir}/system_status
        echo "end print $p_runc stack info" | tee -a ${diagnose_dir}/system_status
    done
}


daemon_status() {
     run systemctl status docker -l | tee -a ${daemon_status_dir}/docker_status
     run systemctl status containerd -l | tee -a ${daemon_status_dir}/containerd_status
     run systemctl status container-storaged -l | tee -a ${daemon_status_dir}/container-storaged_status
     run systemctl status kubelet -l | tee -a ${daemon_status_dir}/kubelet_status
     run systemctl status edge-hub -l | tee -a ${daemon_status_dir}/edgehub_status
}

docker_status() {
    if command -v docker > /dev/null 2>&1; then
        echo "check dockerd process"
        if [ "$is_ps_hang" == "false" ]; then
            run ps -ef|grep -E 'dockerd|docker daemon'|grep -v grep| tee -a ${daemon_status_dir}/docker_status
        else
            echo "ps -ef command hang, skip [ps -ef|grep -E 'dockerd|docker daemon'] check" | tee -a ${daemon_status_dir}/docker_status
        fi

        #docker info
        run docker info | tee -a ${daemon_status_dir}/docker_status
        run docker version | tee -a ${daemon_status_dir}/docker_status
        sudo kill -SIGUSR1 $(cat /var/run/docker.pid)
        cp /var/run/docker/libcontainerd/containerd/events.log ${daemon_status_dir}/containerd_events.log
        sleep 10
        cp /var/run/docker/*.log ${daemon_status_dir}
        cp /etc/docker/daemon.json ${daemon_status_dir}/docker_daemon.json
    fi
}

pouch_status(){
  if command -v pouch > /dev/null 2>&1; then
    echo "check pouch process"
    run ps -ef|grep -E 'pouchd'|grep -v grep| tee -a ${daemon_status_dir}/pouch_status
    run pouch info | tee -a ${daemon_status_dir}/pouch_status
    run pouch version | tee -a ${daemon_status_dir}/pouch_status
  fi
}

showlog() {
    local file=$1
    if [ -f "$file" ]; then
        tail -n 200 $file
    fi
}

#collect log
common_logs() {
    log_tail_lines=10000
    mkdir -p ${diagnose_dir}/logs
    run dmesg -T | tail -n ${log_tail_lines}  | tee ${diagnose_dir}/logs/dmesg.log
    tail -c 500M /var/log/messages &> ${diagnose_dir}/logs/messages
    
    cat /var/log/messages | grep ens-cmd | tail -n ${log_tail_lines} |  tee -a ${diagnose_dir}/logs/ens-logs
    pidof systemd && journalctl -n ${log_tail_lines} -u kubelet | tee -a ${diagnose_dir}/logs/kubelet.log
    pidof systemd && journalctl -n ${log_tail_lines} -u edge-hub | tee -a ${diagnose_dir}/logs/edgehub.log
    if command -v pouch > /dev/null 2>&1 ; then
        pidof systemd && journalctl -n ${log_tail_lines} -u pouch.service &> ${diagnose_dir}/logs/pouch.log || cp /var/log/pouch ${diagnose_dir}/logs/pouch.log
    else
        pidof systemd && journalctl -n ${log_tail_lines} -u docker.service &> ${diagnose_dir}/logs/docker.log || tail -n ${log_tail_lines} /var/log/upstart/docker.log &> ${diagnose_dir}/logs/docker.log
        pidof systemd && journalctl -n ${log_tail_lines} -u containerd.service &> ${diagnose_dir}/logs/containerd.log
    fi
}

#kubelet status
kubelet_status(){
  echo "check kubelet process"
  run ps -ef | grep -E 'kubelet' | grep -v grep | tee -a ${daemon_status_dir}/kubelet_status
  run kubelet --version | tee -a ${daemon_status_dir}/kubelet_status
  cp /etc/systemd/system/kubelet.service.d/10-kubeadm.conf ${daemon_status_dir}/
  cp /etc/systemd/system/kubelet.service ${daemon_status_dir}/
}

#edgehub status
edgehub_status(){
  echo "check edge-hub process"
  run ps -ef | grep -E 'edge-hub' | grep -v grep | tee -a ${daemon_status_dir}/edgehub_status
  run edge-hub --version | tee -a ${daemon_status_dir}/edgehub_status
  cp /etc/systemd/system/edge-hub.service.d/10-edgehub.conf ${daemon_status_dir}/
  cp /etc/systemd/system/edge-hub.service ${daemon_status_dir}/
}

# node cache
node_cache(){
  if ! pgrep -f edge-hub > /dev/null 2>&1; then
    echo "The process edge-hub is not running."
    return 0
  fi

  echo "Get node edge-hub cache date."
  mkdir -p ${diagnose_dir}/node_cache
  cp -r /etc/kubernetes/cache/* ${diagnose_dir}/node_cache/
}

archive() {
    tar -zcvf $tmpdir/diagnose_${timestamp}.tar.gz ${diagnose_dir}
    echo "please get $tmpdir/diagnose_${timestamp}.tar.gz for diagnostics"
}

varlogmessage(){
    grep cloud-init /var/log/messages > $diagnose_dir/varlogmessage.log
}

cluster_dump(){
    kubectl cluster-info dump > $diagnose_dir/cluster_dump.log
}

events(){
    kubectl get events > $diagnose_dir/events.log
}

core_component() {
    local comp="$1"
    local label="$2"
    mkdir -p $diagnose_dir/cs/$comp/
    local pods=`kubectl get -n kube-system po -l $label=$comp | awk '{print $1}'|grep -v NAME`
    for po in ${pods}
    do
        kubectl logs -n kube-system ${po} &> $diagnose_dir/cs/${comp}/${po}.log
    done
}

edge_component() {
  local comp="$1"
  mkdir -p $diagnose_dir/cs/edge
  # try using pouch
  if command -v pouch > /dev/null 2>&1; then
    run pouch ps -a | grep ${comp} | tee -a $diagnose_dir/cs/edge/${comp}_pouch.log
    containerIds=`pouch ps -a | grep ${comp} | grep -v pause | awk -F ' ' '{print $1}'`
    if [[ ! -z $containerIds ]]; then
      for id in ${containerIds[*]}
      do
        run pouch logs ${id} 2>&1 | tee -a $diagnose_dir/cs/edge/${comp}_pouch.log
      done
    fi
    return
  fi

  # try using docker
  run docker ps -a | grep ${comp} | tee -a $diagnose_dir/cs/edge/${comp}_docker.log
  containerIds=`docker ps -a | grep ${comp} | grep -v pause | awk -F ' ' '{print $1}'`
  if [[ ! -z $containerIds ]]; then
    for id in ${containerIds[*]}
    do
      run docker logs ${id} 2>&1 | tee -a $diagnose_dir/cs/edge/${comp}_docker.log
    done
  fi

  #try using containerd
  run crictl ps -a | grep ${comp} | tee -a $diagnose_dir/cs/edge/${comp}_containerd.log
  containerIds=`crictl ps -a | grep ${comp} | awk -F ' ' '{print $1}'`
  if [[ ! -z $containerIds ]]; then
    for id in ${containerIds[*]}
    do
      run crictl logs ${id} 2>&1 | tee -a $diagnose_dir/cs/edge/${comp}_containerd.log
    done
  fi
}

etcd() {
    journalctl -u etcd -xe &> $diagnose_dir/cs/etcd.log
}

edgeadm_logs() {
  cp /var/log/edgeadm.log ${diagnose_dir}/logs
}

storageplugins() {
    mkdir -p ${diagnose_dir}/storage/
    cp /var/log/alicloud/* ${diagnose_dir}/storage/
}

sandbox_runtime_status() {
    if [[ ! -z $(pidof dockerd) || -z $(pidof containerd) ]]; then
        if ! grep -q 'io.containerd.rund.v2' /etc/containerd/config.toml; then
            return 0
        fi
    fi
    wget --connect-timeout=3 http://aliacs-k8s-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com/public/diagnose/sandbox-runtime-status.tgz -q -O ${diagnose_dir}/sandbox-runtime-status.tgz
    tar -xzvf ${diagnose_dir}/sandbox-runtime-status.tgz -C ${diagnose_dir}
    pushd ${diagnose_dir}/sandbox-runtime-status
    bash script_collect.sh >> $diagnose_dir/sandbox_runtime.status
    popd
}

upload_oss() {
  if [[ "$UPLOAD_OSS" == "" ]]; then
      return 0
  fi

  bucket_path=${UPLOAD_OSS}
  diagnose_file=$tmpdir/diagnose_${timestamp}.tar.gz

  if ! command_exists ossutil; then
    curl -o /usr/local/bin/ossutil http://gosspublic.alicdn.com/ossutil/1.6.10/ossutil64
    chmod u+x /usr/local/bin/ossutil
  fi


  region=$(curl --retry 10 --retry-delay 5 http://100.100.100.200/latest/meta-data/region-id)
  endpoint="oss-$region.aliyuncs.com"
  if [[ "$ACCESS_KEY_ID" == "" ]]; then
    roleName=$(curl --retry 10 --retry-delay 5 100.100.100.200/latest/meta-data/ram/security-credentials/)
    echo "
[Credentials]
        language = CH
        endpoint = $endpoint
[AkService]
        ecsAk=http://100.100.100.200/latest/meta-data/Ram/security-credentials/$roleName" > ./config
  else
    echo "
[Credentials]
        language = CH
        endpoint = $endpoint
        accessKeyID = $ACCESS_KEY_ID
        accessKeySecret = $ACCESS_KEY_SECRET
" > ./config
  fi
  bucket_name=${bucket_path%%/*}
  oss_endpoint=$(ossutil stat oss://$bucket_name --config-file ./config | grep ExtranetEndpoint | awk '{print $3}')
  if [[ "$oss_endpoint" != "" ]]; then
    endpoint=$oss_endpoint
  fi
  ossutil cp ./${diagnose_file} oss://$bucket_path/$diagnose_file --config-file ./config --endpoint $endpoint

  if [[ "$OSS_PUBLIC_LINK" != "" ]]; then
    ossutil sign --timeout 7200 oss://$bucket_path/$diagnose_file --config-file ./config --endpoint $endpoint
  fi
}

parse_args() {
    while
        [[ $# -gt 0 ]]
    do
        key="$1"

        case $key in
        --oss)
            export UPLOAD_OSS=$2
            shift
            ;;
        --oss-public-link)
            export OSS_PUBLIC_LINK="true"
            ;;
        --access-key-id)
            export ACCESS_KEY_ID=$2
            shift
            ;;
        --access-key-secret)
            export ACCESS_KEY_SECRET=$2
            shift
            ;;
        *)
            echo "unknown option [$key]"
            ;;
        esac
        shift
    done
}

pd_collect() {
    os_env
    system_info
    service_status
    network_info
    check_ps_hang
    system_status
    daemon_status
    docker_status
    pouch_status
    sandbox_runtime_status
    common_logs

    # memory
    memory_info

    # edge 
    edge_component "kube-proxy"
    edge_component "edge-hub"
    edge_component "cloud-hub"
    edge_component "tunnel-agent"
    edge_component "logtail"
    edge_component "edge-tunnel-server"
    edge_component "manager"
    edge_component "raven-agent" 
    edge_component "flannel"
    edge_component "coredns"
    kubelet_status
    edgehub_status
    edgeadm_logs
    node_cache

    varlogmessage
    core_component "cloud-controller-manager" "app"
    core_component "kube-apiserver" "component"
    core_component "kube-controller-manager" "component"
    core_component "kube-scheduler" "component"
    events
    storageplugins
    etcd
    cluster_dump
    archive
}

parse_args "$@"

pd_collect

upload_oss

echo "请上传 $tmpdir/diagnose_${timestamp}.tar.gz"
