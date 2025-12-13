cd "$(dirname "$0")"
echo "=====check pouch config====="
bash check_config_with_pouch_rpm.sh
if [ $? -eq 0 ]; then
        echo "the pouch config is valid"
fi

echo "======check containerd config====="
bash check_config_with_containerd_rpm.sh
if [ $? -eq 0 ]; then
    echo "the containerd config is valid"
fi

echo -e "\n=====check if the current fd number has reached the maximum value====="
bash check_host_fd_exceed.sh
if [ $? -eq 0 ]; then
    echo "fd number is ok"
fi

echo -e "\n======check if containerd-shim leaked====="
bash check_containerd_shim_leak.sh
echo "$? containerd-shim leaked"

echo -e "\n=====check if kata-proxy leaked====="
bash check_kata_proxy_leak.sh
echo "$? kata-proxy leaked"

echo -e "\n=====check if netns leaked====="
bash check_netns_leak.sh
echo "$? netns leaked"

echo -e "\n=====check if qemu leaked====="
bash check_qemu_leak.sh
echo "$? qemus leaked"

echo -e "\n=====check if snapshots leaked====="
bash check_snapshot_leak.sh
echo "$? snapshots leaked"

echo -e "\n=====check if shim alive====="
bash check_shim_alive.sh
echo "$? shims blocked"
