#!/usr/bin/env bash

################################################################################
# kudig.sh - Kubernetes节点诊断日志分析工具
# 
# 功能：分析 diagnose_k8s.sh 收集的诊断日志，识别异常并输出中英文报告
# 
# 使用方法:
#   ./kudig.sh <diagnose_dir>              # 分析指定诊断目录
#   ./kudig.sh --json <diagnose_dir>       # 输出JSON格式
#   ./kudig.sh --verbose <diagnose_dir>    # 详细模式
#   ./kudig.sh --help                      # 显示帮助信息
#
# 示例:
#   ./kudig.sh /tmp/diagnose_1702468800
#   ./kudig.sh --json /tmp/diagnose_1702468800 > report.json
#
# 作者: kudig.sh Team
# 版本: 1.0.0
################################################################################

set -euo pipefail

# ============================================================================
# 全局变量定义
# ============================================================================

VERSION="1.0.0"
SCRIPT_NAME=$(basename "$0")
DIAGNOSE_DIR=""
OUTPUT_FORMAT="text"  # text, json
VERBOSE=false
OUTPUT_FILE=""

# 异常数组 - 格式: "严重级别|中文名称|英文标识|详情|位置"
declare -a ANOMALIES=()

# 颜色定义（用于终端输出）
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    GREEN='\033[0;32m'
    NC='\033[0m' # No Color
else
    RED=''
    YELLOW=''
    BLUE=''
    GREEN=''
    NC=''
fi

# 严重级别常量
SEVERITY_CRITICAL="严重"
SEVERITY_WARNING="警告"
SEVERITY_INFO="提示"

# ============================================================================
# 工具函数
# ============================================================================

# 打印使用说明
usage() {
    cat << EOF
用法: $SCRIPT_NAME [选项] <诊断目录>

Kubernetes节点诊断日志分析工具
分析 diagnose_k8s.sh 收集的诊断数据，识别异常并生成报告

选项:
    -h, --help              显示此帮助信息
    -v, --version           显示版本信息
    --verbose               详细输出模式
    --json                  输出JSON格式
    -o, --output <文件>     保存报告到指定文件

参数:
    <诊断目录>              diagnose_k8s.sh 生成的诊断目录路径

示例:
    $SCRIPT_NAME /tmp/diagnose_1702468800
    $SCRIPT_NAME --json /tmp/diagnose_1702468800 > report.json
    $SCRIPT_NAME --verbose -o report.txt /tmp/diagnose_1702468800

EOF
    exit 0
}

# 打印版本信息
version() {
    echo "$SCRIPT_NAME version $VERSION"
    exit 0
}

# 日志输出函数
log_info() {
    if [[ "$VERBOSE" == true ]]; then
        echo -e "${BLUE}[INFO]${NC} $*" >&2
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_debug() {
    if [[ "$VERBOSE" == true ]]; then
        echo -e "[DEBUG] $*" >&2
    fi
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 检查必要的命令
check_required_commands() {
    local required_cmds=("grep" "awk" "sed" "wc" "sort" "uniq" "tail" "head" "find")
    local missing_cmds=()
    
    for cmd in "${required_cmds[@]}"; do
        if ! command_exists "$cmd"; then
            missing_cmds+=("$cmd")
        fi
    done
    
    if [[ ${#missing_cmds[@]} -gt 0 ]]; then
        log_error "缺少必要的命令: ${missing_cmds[*]}"
        log_error "请安装这些命令后重试"
        exit 1
    fi
    
    log_info "环境检查通过"
}

# 验证诊断目录
validate_diagnose_dir() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log_error "诊断目录不存在: $dir"
        exit 1
    fi
    
    log_info "验证诊断目录: $dir"
    
    # 检查关键文件是否存在（宽松检查，部分文件缺失也可以继续）
    local key_files=("system_info" "service_status" "system_status")
    local found_files=0
    
    for file in "${key_files[@]}"; do
        if [[ -f "$dir/$file" ]]; then
            ((found_files++))
        fi
    done
    
    if [[ $found_files -eq 0 ]]; then
        log_warn "诊断目录结构可能不完整，未找到关键文件"
        log_warn "将尝试分析现有文件..."
    else
        log_info "找到 $found_files 个关键文件"
    fi
}

# 添加异常到数组
# 参数: 严重级别 中文名称 英文标识 详情 位置
add_anomaly() {
    local severity="$1"
    local cn_name="$2"
    local en_name="$3"
    local details="$4"
    local location="$5"
    
    ANOMALIES+=("$severity|$cn_name|$en_name|$details|$location")
    log_debug "检测到异常: $en_name - $cn_name"
}

# 安全读取文件（如果文件不存在返回空）
safe_cat() {
    local file="$1"
    if [[ -f "$file" ]]; then
        cat "$file" 2>/dev/null || true
    fi
}

# 安全获取文件行数
safe_line_count() {
    local file="$1"
    if [[ -f "$file" ]]; then
        wc -l < "$file" 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

# 从文件中提取数值
extract_number() {
    local file="$1"
    local pattern="$2"
    
    if [[ -f "$file" ]]; then
        grep -oP "$pattern" "$file" 2>/dev/null | head -1 || echo "0"
    else
        echo "0"
    fi
}

# ============================================================================
# 参数解析
# ============================================================================

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -v|--version)
                version
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --json)
                OUTPUT_FORMAT="json"
                shift
                ;;
            -o|--output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            -*)
                log_error "未知选项: $1"
                usage
                ;;
            *)
                if [[ -z "$DIAGNOSE_DIR" ]]; then
                    DIAGNOSE_DIR="$1"
                else
                    log_error "只能指定一个诊断目录"
                    usage
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$DIAGNOSE_DIR" ]]; then
        log_error "请指定诊断目录"
        usage
    fi
}

# ============================================================================
# 文件解析辅助函数
# ============================================================================

# 从system_info中提取CPU核心数
get_cpu_cores() {
    local system_info="$DIAGNOSE_DIR/system_info"
    if [[ -f "$system_info" ]]; then
        # 尝试从 /proc/cpuinfo 或 lscpu 输出中提取
        local cores=$(grep -c "^processor" "$system_info" 2>/dev/null || echo "0")
        if [[ $cores -eq 0 ]]; then
            cores=$(grep -oP 'CPU\(s\):\s*\K\d+' "$system_info" 2>/dev/null | head -1 || echo "4")
        fi
        echo "$cores"
    else
        echo "4"  # 默认值
    fi
}

# 从system_info中提取总内存（KB）
get_total_memory() {
    local memory_info="$DIAGNOSE_DIR/memory_info"
    if [[ -f "$memory_info" ]]; then
        local mem_kb=$(grep -oP 'MemTotal:\s*\K\d+' "$memory_info" 2>/dev/null | head -1 || echo "0")
        echo "$mem_kb"
    else
        echo "0"
    fi
}

# 从system_status提取负载信息
get_load_average() {
    local system_status="$DIAGNOSE_DIR/system_status"
    if [[ -f "$system_status" ]]; then
        # 从uptime输出中提取负载
        grep -oP 'load average:\s*\K[\d.]+,\s*[\d.]+,\s*[\d.]+' "$system_status" 2>/dev/null | head -1 || echo "0,0,0"
    else
        echo "0,0,0"
    fi
}

# 解析负载字符串，返回指定位置的值（1=1min, 2=5min, 3=15min）
parse_load() {
    local load_str="$1"
    local pos="$2"
    echo "$load_str" | awk -F',' -v p="$pos" '{gsub(/ /,"",$p); print $p}'
}

# 从df输出中检查磁盘使用率
check_disk_usage() {
    local system_status="$DIAGNOSE_DIR/system_status"
    if [[ ! -f "$system_status" ]]; then
        return
    fi
    
    # 提取df -h的输出部分
    awk '/^-+run df -h/,/^-+End of df/' "$system_status" 2>/dev/null | \
        grep -E '^/' | \
        awk '{gsub(/%/,"",$(NF-1)); if($(NF-1)+0 >= 90) print $0}'
}

# 从service_status提取服务状态
get_service_status() {
    local service_name="$1"
    local service_status="$DIAGNOSE_DIR/service_status"
    
    if [[ ! -f "$service_status" ]]; then
        echo "unknown"
        return
    fi
    
    # 查找服务状态
    if grep -q "${service_name}.*running" "$service_status" 2>/dev/null; then
        echo "running"
    elif grep -q "${service_name}.*active" "$service_status" 2>/dev/null; then
        echo "running"
    elif grep -q "${service_name}.*stopped" "$service_status" 2>/dev/null; then
        echo "stopped"
    elif grep -q "${service_name}.*inactive" "$service_status" 2>/dev/null; then
        echo "stopped"
    elif grep -q "${service_name}.*failed" "$service_status" 2>/dev/null; then
        echo "failed"
    else
        echo "unknown"
    fi
}

# 从daemon_status中获取详细服务状态
get_daemon_status() {
    local daemon_name="$1"
    local daemon_file="$DIAGNOSE_DIR/daemon_status/${daemon_name}_status"
    
    if [[ ! -f "$daemon_file" ]]; then
        echo "unknown"
        return
    fi
    
    if grep -qi "Active:.*running" "$daemon_file" 2>/dev/null; then
        echo "running"
    elif grep -qi "Active:.*active" "$daemon_file" 2>/dev/null; then
        echo "running"
    elif grep -qi "Active:.*failed" "$daemon_file" 2>/dev/null; then
        echo "failed"
    elif grep -qi "Active:.*inactive" "$daemon_file" 2>/dev/null; then
        echo "stopped"
    else
        echo "unknown"
    fi
}

# 从日志文件中统计错误模式出现次数
count_pattern_in_log() {
    local log_file="$1"
    local pattern="$2"
    
    if [[ ! -f "$log_file" ]]; then
        echo "0"
        return
    fi
    
    grep -c "$pattern" "$log_file" 2>/dev/null || echo "0"
}

# 检查文件中是否存在模式
pattern_exists() {
    local file="$1"
    local pattern="$2"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    grep -q "$pattern" "$file" 2>/dev/null
}

# 从网络信息中提取连接跟踪表信息
get_conntrack_info() {
    local network_info="$DIAGNOSE_DIR/network_info"
    if [[ ! -f "$network_info" ]]; then
        echo "0:0"
        return
    fi
    
    # 统计当前连接数
    local current=$(grep -c '^ipv4' "$network_info" 2>/dev/null || echo "0")
    
    # 尝试从sysctl中获取最大值
    local system_info="$DIAGNOSE_DIR/system_info"
    local max=$(grep 'net.netfilter.nf_conntrack_max' "$system_info" 2>/dev/null | awk '{print $NF}' | head -1 || echo "65536")
    
    echo "$current:$max"
}

# ============================================================================
# 异常检测器实现
# ============================================================================

# 检测系统资源异常
check_system_resources() {
    log_info "检查系统资源..."
    
    # 1. 检查CPU负载
    local load_avg=$(get_load_average)
    local cpu_cores=$(get_cpu_cores)
    local load_15min=$(parse_load "$load_avg" 3)
    
    # 负载阈值：CPU核心数的4倍
    local load_threshold=$(awk "BEGIN {print $cpu_cores * 4}")
    
    if (( $(awk "BEGIN {print ($load_15min > $load_threshold)}") )); then
        add_anomaly "$SEVERITY_CRITICAL" \
            "系统负载过高" \
            "HIGH_SYSTEM_LOAD" \
            "15分钟平均负载 $load_15min，超过CPU核心数($cpu_cores)的4倍" \
            "system_status"
    elif (( $(awk "BEGIN {print ($load_15min > $cpu_cores * 2)}") )); then
        add_anomaly "$SEVERITY_WARNING" \
            "系统负载偏高" \
            "ELEVATED_SYSTEM_LOAD" \
            "15分钟平均负载 $load_15min，超过CPU核心数($cpu_cores)的2倍" \
            "system_status"
    fi
    
    # 2. 检查内存压力
    local memory_info="$DIAGNOSE_DIR/memory_info"
    if [[ -f "$memory_info" ]]; then
        local total_mem=$(get_total_memory)
        local avail_mem=$(grep -oP 'MemAvailable:\s*\K\d+' "$memory_info" 2>/dev/null | head -1 || echo "0")
        
        if [[ $total_mem -gt 0 && $avail_mem -gt 0 ]]; then
            local mem_usage_percent=$(awk "BEGIN {printf \"%.0f\", (1 - $avail_mem / $total_mem) * 100}")
            
            if [[ $mem_usage_percent -ge 95 ]]; then
                add_anomaly "$SEVERITY_CRITICAL" \
                    "内存使用率过高" \
                    "HIGH_MEMORY_USAGE" \
                    "内存使用率 ${mem_usage_percent}%，可能导致OOM" \
                    "memory_info"
            elif [[ $mem_usage_percent -ge 85 ]]; then
                add_anomaly "$SEVERITY_WARNING" \
                    "内存使用率偏高" \
                    "ELEVATED_MEMORY_USAGE" \
                    "内存使用率 ${mem_usage_percent}%" \
                    "memory_info"
            fi
        fi
    fi
    
    # 3. 检查磁盘空间
    local high_disk_usage=$(check_disk_usage)
    if [[ -n "$high_disk_usage" ]]; then
        while IFS= read -r line; do
            local usage=$(echo "$line" | awk '{print $(NF-1)}')
            local mount=$(echo "$line" | awk '{print $NF}')
            
            if [[ $usage -ge 95 ]]; then
                add_anomaly "$SEVERITY_CRITICAL" \
                    "磁盘空间严重不足" \
                    "DISK_SPACE_CRITICAL" \
                    "挂载点 $mount 使用率 ${usage}%" \
                    "system_status"
            elif [[ $usage -ge 90 ]]; then
                add_anomaly "$SEVERITY_WARNING" \
                    "磁盘空间不足" \
                    "DISK_SPACE_LOW" \
                    "挂载点 $mount 使用率 ${usage}%" \
                    "system_status"
            fi
        done <<< "$high_disk_usage"
    fi
    
    # 4. 检查文件句柄
    local system_status="$DIAGNOSE_DIR/system_status"
    if [[ -f "$system_status" ]]; then
        # 查找文件句柄最多的进程
        local max_fds=$(awk '/fds \(PID/{print $1}' "$system_status" 2>/dev/null | head -1 || echo "0")
        
        # 获取系统限制
        local system_info="$DIAGNOSE_DIR/system_info"
        local max_files=$(grep 'fs.file-max' "$system_info" 2>/dev/null | awk '{print $NF}' | head -1 || echo "1000000")
        
        if [[ $max_fds -gt 50000 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "文件句柄使用量过高" \
                "HIGH_FILE_HANDLES" \
                "进程最大文件句柄数: $max_fds" \
                "system_status"
        fi
    fi
    
    # 5. 检查PID泄漏
    if [[ -f "$system_status" ]]; then
        # 查找线程数最多的进程
        local max_threads=$(awk '/^-+start pid leak detect/,/^-+done pid leak detect/{if($1~/^[0-9]+$/) print $1}' "$system_status" 2>/dev/null | tail -1 || echo "0")
        
        if [[ $max_threads -gt 10000 ]]; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "进程/线程数异常" \
                "PID_LEAK_DETECTED" \
                "某进程线程数达到 $max_threads" \
                "system_status"
        elif [[ $max_threads -gt 5000 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "进程/线程数偏高" \
                "HIGH_THREAD_COUNT" \
                "某进程线程数达到 $max_threads" \
                "system_status"
        fi
    fi
    
    # 6. 检查inode使用率（从df -i输出）
    if [[ -f "$system_status" ]]; then
        local high_inode=$(awk '/^-+run df/,/^-+End of df/' "$system_status" 2>/dev/null | \
            grep -E '^/' | \
            awk '{gsub(/%/,"",$(NF-2)); if($(NF-2)+0 >= 90) print $(NF-2), $NF}')
        
        if [[ -n "$high_inode" ]]; then
            while IFS= read -r line; do
                local usage=$(echo "$line" | awk '{print $1}')
                local mount=$(echo "$line" | awk '{print $2}')
                
                add_anomaly "$SEVERITY_WARNING" \
                    "Inode使用率过高" \
                    "HIGH_INODE_USAGE" \
                    "挂载点 $mount 的inode使用率 ${usage}%" \
                    "system_status"
            done <<< "$high_inode"
        fi
    fi
}

# 检测进程与服务异常
check_process_services() {
    log_info "检查进程与服务..."
    
    # 1. 检查kubelet服务状态
    local kubelet_status=$(get_daemon_status "kubelet")
    if [[ "$kubelet_status" == "failed" ]]; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "Kubelet服务未运行" \
            "KUBELET_SERVICE_DOWN" \
            "kubelet.service状态为failed" \
            "daemon_status/kubelet_status"
    elif [[ "$kubelet_status" == "stopped" ]]; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "Kubelet服务停止" \
            "KUBELET_SERVICE_STOPPED" \
            "kubelet.service未启动" \
            "daemon_status/kubelet_status"
    fi
    
    # 2. 检查容器运行时服务（docker或containerd）
    local docker_status=$(get_daemon_status "docker")
    local containerd_status=$(get_daemon_status "containerd")
    
    if [[ "$docker_status" == "failed" && "$containerd_status" == "failed" ]]; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "容器运行时服务异常" \
            "CONTAINER_RUNTIME_DOWN" \
            "docker和containerd服务均为failed状态" \
            "daemon_status/"
    elif [[ "$docker_status" == "stopped" && "$containerd_status" == "stopped" ]]; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "容器运行时服务停止" \
            "CONTAINER_RUNTIME_STOPPED" \
            "docker和containerd服务均未启动" \
            "daemon_status/"
    fi
    
    # 3. 检查ps命令是否挂起
    local ps_status="$DIAGNOSE_DIR/ps_command_status"
    if [[ -f "$ps_status" ]] && pattern_exists "$ps_status" "ps -ef command is hung"; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "ps命令挂起" \
            "PS_COMMAND_HUNG" \
            "ps -ef命令挂起，系统可能存在D状态进程" \
            "ps_command_status"
    fi
    
    # 4. 检查D状态进程
    if [[ -f "$ps_status" ]] && pattern_exists "$ps_status" "process.*is in State D"; then
        local d_proc_count=$(grep -c "is in State D" "$ps_status" 2>/dev/null || echo "0")
        
        add_anomaly "$SEVERITY_CRITICAL" \
            "存在D状态进程" \
            "PROCESS_IN_D_STATE" \
            "检测到 $d_proc_count 个不可中断睡眠状态的进程" \
            "ps_command_status"
    fi
    
    # 5. 检查runc进程挂起
    local system_status="$DIAGNOSE_DIR/system_status"
    if [[ -f "$system_status" ]] && pattern_exists "$system_status" "runc process.*maybe hang"; then
        local runc_count=$(grep -c "runc process.*maybe hang" "$system_status" 2>/dev/null || echo "0")
        
        add_anomaly "$SEVERITY_WARNING" \
            "runc进程可能挂起" \
            "RUNC_PROCESS_HANG" \
            "检测到 $runc_count 个runc进程可能处于挂起状态" \
            "system_status"
    fi
    
    # 6. 检查关键服务状态
    local service_status="$DIAGNOSE_DIR/service_status"
    if [[ -f "$service_status" ]]; then
        # 检查firewalld是否运行（Kubernetes节点应关闭）
        if pattern_exists "$service_status" "firewalld.*running"; then
            add_anomaly "$SEVERITY_WARNING" \
                "Firewalld服务运行中" \
                "FIREWALLD_RUNNING" \
                "Kubernetes节点建议关闭firewalld服务" \
                "service_status"
        fi
    fi
}

# 检测网络异常
check_network() {
    log_info "检查网络状态..."
    
    # 1. 检查连接跟踪表
    local conntrack_info=$(get_conntrack_info)
    IFS=':' read -r current_conn max_conn <<< "$conntrack_info"
    
    if [[ $max_conn -gt 0 ]]; then
        local usage_percent=$(awk "BEGIN {printf \"%.0f\", $current_conn / $max_conn * 100}")
        
        if [[ $usage_percent -ge 95 ]]; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "连接跟踪表满" \
                "CONNTRACK_TABLE_FULL" \
                "当前连接数 $current_conn/$max_conn (${usage_percent}%)，接近上限" \
                "network_info"
        elif [[ $usage_percent -ge 80 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "连接跟踪表使用率高" \
                "CONNTRACK_TABLE_HIGH_USAGE" \
                "当前连接数 $current_conn/$max_conn (${usage_percent}%)" \
                "network_info"
        fi
    fi
    
    # 2. 检查网卡状态
    local network_info="$DIAGNOSE_DIR/network_info"
    if [[ -f "$network_info" ]]; then
        # 查找down状态的网卡（排除lo和veth）
        local down_interfaces=$(awk '/state DOWN/{print $2}' "$network_info" 2>/dev/null | \
            grep -v '^lo' | grep -v '^veth' | sed 's/:$//')
        
        if [[ -n "$down_interfaces" ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "网卡接口down" \
                "NETWORK_INTERFACE_DOWN" \
                "以下网卡处于down状态: $down_interfaces" \
                "network_info"
        fi
    fi
    
    # 3. 检查路由表
    if [[ -f "$network_info" ]]; then
        # 检查是否有默认路由
        if ! pattern_exists "$network_info" "default via"; then
            add_anomaly "$SEVERITY_WARNING" \
                "缺少默认路由" \
                "NO_DEFAULT_ROUTE" \
                "未检测到默认路由配置" \
                "network_info"
        fi
    fi
    
    # 4. 检查端口监听
    local system_status="$DIAGNOSE_DIR/system_status"
    if [[ -f "$system_status" ]]; then
        # 检查kubelet端口（10250）
        if ! pattern_exists "$system_status" ":10250.*LISTEN"; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "Kubelet端口未监听" \
                "KUBELET_PORT_NOT_LISTENING" \
                "10250端口未处于监听状态" \
                "system_status"
        fi
    fi
    
    # 5. 检查iptables规则数量
    if [[ -f "$network_info" ]]; then
        local iptables_rules=$(grep -c '^-A' "$network_info" 2>/dev/null || echo "0")
        
        if [[ $iptables_rules -gt 50000 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "iptables规则过多" \
                "TOO_MANY_IPTABLES_RULES" \
                "iptables规则数量: $iptables_rules，可能影响性能" \
                "network_info"
        fi
    fi
}

# 检测内核异常
check_kernel() {
    log_info "检查内核状态..."
    
    # 1. 检查内核panic
    local dmesg_log="$DIAGNOSE_DIR/logs/dmesg.log"
    if [[ -f "$dmesg_log" ]] && pattern_exists "$dmesg_log" "Kernel panic"; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "内核Panic" \
            "KERNEL_PANIC" \
            "内核发生panic事件" \
            "logs/dmesg.log"
    fi
    
    # 2. 检查OOM Killer
    if [[ -f "$dmesg_log" ]]; then
        local oom_count=$(count_pattern_in_log "$dmesg_log" "Out of memory: Kill process")
        
        if [[ $oom_count -gt 0 ]]; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "内核触发OOM杀进程" \
                "KERNEL_OOM_KILLER" \
                "内核OOM Killer被触发 $oom_count 次" \
                "logs/dmesg.log"
        fi
    fi
    
    # 从logs/messages中也检查OOM
    local messages_log="$DIAGNOSE_DIR/logs/messages"
    if [[ -f "$messages_log" ]]; then
        local oom_count_msg=$(count_pattern_in_log "$messages_log" "Out of memory")
        
        if [[ $oom_count_msg -gt 0 ]]; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "系统内存不足" \
                "SYSTEM_OUT_OF_MEMORY" \
                "系统日志显示内存不足 $oom_count_msg 次" \
                "logs/messages"
        fi
    fi
    
    # 3. 检查文件系统错误
    if [[ -f "$dmesg_log" ]]; then
        # 检查只读文件系统
        if pattern_exists "$dmesg_log" "Read-only file system"; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "文件系统只读" \
                "FILESYSTEM_READONLY" \
                "文件系统被重新挂载为只读模式" \
                "logs/dmesg.log"
        fi
        
        # 检查IO错误
        local io_error_count=$(count_pattern_in_log "$dmesg_log" "I/O error")
        if [[ $io_error_count -gt 10 ]]; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "磁盘IO错误" \
                "DISK_IO_ERROR" \
                "检测到 $io_error_count 次IO错误" \
                "logs/dmesg.log"
        fi
    fi
    
    # 4. 检查内核模块加载失败
    if [[ -f "$dmesg_log" ]] && pattern_exists "$dmesg_log" "module.*failed"; then
        add_anomaly "$SEVERITY_WARNING" \
            "内核模块加载失败" \
            "KERNEL_MODULE_LOAD_FAILED" \
            "存在内核模块加载失败" \
            "logs/dmesg.log"
    fi
    
    # 5. 检查NMI watchdog
    if [[ -f "$dmesg_log" ]] && pattern_exists "$dmesg_log" "NMI watchdog"; then
        add_anomaly "$SEVERITY_WARNING" \
            "NMI Watchdog触发" \
            "NMI_WATCHDOG_TRIGGERED" \
            "硬件看门狗被触发" \
            "logs/dmesg.log"
    fi
}

# 检测容器运行时异常
check_container_runtime() {
    log_info "检查容器运行时..."
    
    # 1. 检查Docker日志中的错误
    local docker_log="$DIAGNOSE_DIR/logs/docker.log"
    if [[ -f "$docker_log" ]]; then
        # 检查Docker启动失败
        if pattern_exists "$docker_log" "Failed to start"; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "Docker启动失败" \
                "DOCKER_START_FAILED" \
                "Docker服务启动失败" \
                "logs/docker.log"
        fi
        
        # 检查存储驱动错误
        if pattern_exists "$docker_log" "storage driver.*error"; then
            add_anomaly "$SEVERITY_CRITICAL" \
                "Docker存储驱动错误" \
                "DOCKER_STORAGE_DRIVER_ERROR" \
                "Docker存储驱动出现错误" \
                "logs/docker.log"
        fi
    fi
    
    # 2. 检查Containerd日志
    local containerd_log="$DIAGNOSE_DIR/logs/containerd.log"
    if [[ -f "$containerd_log" ]]; then
        # 检查容器创建失败
        local create_failed=$(count_pattern_in_log "$containerd_log" "failed to create")
        if [[ $create_failed -gt 10 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "容器创建失败率高" \
                "CONTAINER_CREATE_FAILED" \
                "容器创建失败 $create_failed 次" \
                "logs/containerd.log"
        fi
    fi
    
    # 3. 检查镜像拉取失败
    local kubelet_log="$DIAGNOSE_DIR/logs/kubelet.log"
    if [[ -f "$kubelet_log" ]]; then
        local pull_failed=$(count_pattern_in_log "$kubelet_log" "Failed to pull image")
        if [[ $pull_failed -gt 5 ]]; then
            add_anomaly "$SEVERITY_WARNING" \
                "镜像拉取失败" \
                "IMAGE_PULL_FAILED" \
                "镜像拉取失败 $pull_failed 次" \
                "logs/kubelet.log"
        fi
    fi
    
    # 4. 检查runc挂起（已在check_process_services中检查）
    # 这里不重复检测
}

# 检测Kubernetes组件异常
check_kubernetes() {
    log_info "检查Kubernetes组件..."
    
    local kubelet_log="$DIAGNOSE_DIR/logs/kubelet.log"
    
    if [[ ! -f "$kubelet_log" ]]; then
        return
    fi
    
    # 1. 检查PLEG不健康
    if pattern_exists "$kubelet_log" "PLEG is not healthy"; then
        local pleg_count=$(count_pattern_in_log "$kubelet_log" "PLEG is not healthy")
        add_anomaly "$SEVERITY_CRITICAL" \
            "Kubelet PLEG不健康" \
            "KUBELET_PLEG_UNHEALTHY" \
            "PLEG（Pod生命周期事件生成器）不健康，出现 $pleg_count 次" \
            "logs/kubelet.log"
    fi
    
    # 2. 检查CNI错误
    if pattern_exists "$kubelet_log" "Failed to create pod sandbox.*CNI"; then
        local cni_error=$(count_pattern_in_log "$kubelet_log" "CNI.*failed")
        add_anomaly "$SEVERITY_CRITICAL" \
            "CNI网络插件错误" \
            "CNI_PLUGIN_ERROR" \
            "CNI网络插件失败 $cni_error 次" \
            "logs/kubelet.log"
    fi
    
    # 3. 检查证书过期
    if pattern_exists "$kubelet_log" "certificate has expired"; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "证书已过期" \
            "CERTIFICATE_EXPIRED" \
            "Kubelet证书已过期" \
            "logs/kubelet.log"
    elif pattern_exists "$kubelet_log" "certificate will expire"; then
        add_anomaly "$SEVERITY_WARNING" \
            "证书即将过期" \
            "CERTIFICATE_EXPIRING" \
            "Kubelet证书即将过期" \
            "logs/kubelet.log"
    fi
    
    # 4. 检查API Server连接失败
    local api_conn_failed=$(count_pattern_in_log "$kubelet_log" "Unable to connect to the server")
    if [[ $api_conn_failed -gt 10 ]]; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "API Server连接失败" \
            "APISERVER_CONNECTION_FAILED" \
            "无法连接到API Server，出现 $api_conn_failed 次" \
            "logs/kubelet.log"
    fi
    
    # 5. 检查认证失败
    if pattern_exists "$kubelet_log" "Unauthorized"; then
        local auth_failed=$(count_pattern_in_log "$kubelet_log" "Unauthorized")
        add_anomaly "$SEVERITY_CRITICAL" \
            "Kubelet认证失败" \
            "KUBELET_AUTH_FAILED" \
            "Kubelet认证失败 $auth_failed 次" \
            "logs/kubelet.log"
    fi
    
    # 6. 检查Pod驱逐事件
    if pattern_exists "$kubelet_log" "evicted pod"; then
        local evicted=$(count_pattern_in_log "$kubelet_log" "evicted pod")
        add_anomaly "$SEVERITY_WARNING" \
            "Pod被驱逐" \
            "POD_EVICTED" \
            "Pod被驱逐 $evicted 次，可能由于资源不足" \
            "logs/kubelet.log"
    fi
    
    # 7. 检查Node NotReady
    if pattern_exists "$kubelet_log" "Node.*NotReady"; then
        add_anomaly "$SEVERITY_CRITICAL" \
            "节点NotReady状态" \
            "NODE_NOT_READY" \
            "节点处于NotReady状态" \
            "logs/kubelet.log"
    fi
    
    # 8. 检查磁盘压力驱逐
    if pattern_exists "$kubelet_log" "DiskPressure"; then
        add_anomaly "$SEVERITY_WARNING" \
            "磁盘压力" \
            "DISK_PRESSURE" \
            "节点存在磁盘压力" \
            "logs/kubelet.log"
    fi
    
    # 9. 检查内存压力驱逐
    if pattern_exists "$kubelet_log" "MemoryPressure"; then
        add_anomaly "$SEVERITY_WARNING" \
            "内存压力" \
            "MEMORY_PRESSURE" \
            "节点存在内存压力" \
            "logs/kubelet.log"
    fi
}

# 检测时间同步异常
check_time_sync() {
    log_info "检查时间同步..."
    
    local service_status="$DIAGNOSE_DIR/service_status"
    
    if [[ ! -f "$service_status" ]]; then
        return
    fi
    
    # 检查ntpd和chronyd服务状态
    local ntpd_status=$(get_service_status "ntpd")
    local chronyd_status=$(get_service_status "chronyd")
    
    if [[ "$ntpd_status" != "running" && "$chronyd_status" != "running" ]]; then
        add_anomaly "$SEVERITY_INFO" \
            "时间同步服务未运行" \
            "TIME_SYNC_SERVICE_DOWN" \
            "ntpd和chronyd服务均未运行" \
            "service_status"
    fi
}

# 检测配置异常
check_configuration() {
    log_info "检查系统配置..."
    
    local system_info="$DIAGNOSE_DIR/system_info"
    
    if [[ ! -f "$system_info" ]]; then
        return
    fi
    
    # 1. 检查swap是否禁用
    if pattern_exists "$system_info" "SwapTotal:.*[1-9]"; then
        # swap不为0，说明没有完全禁用
        local swap_total=$(grep -oP 'SwapTotal:\s*\K\d+' "$system_info" 2>/dev/null | head -1 || echo "0")
        if [[ $swap_total -gt 0 ]]; then
            add_anomaly "$SEVERITY_INFO" \
                "Swap未禁用" \
                "SWAP_NOT_DISABLED" \
                "Kubernetes节点建议禁用swap，当前 ${swap_total}KB" \
                "system_info"
        fi
    fi
    
    # 2. 检查关键的sysctl参数
    # 检查ip_forward
    if pattern_exists "$system_info" "net.ipv4.ip_forward = 0"; then
        add_anomaly "$SEVERITY_WARNING" \
            "IP转发未启用" \
            "IP_FORWARD_DISABLED" \
            "net.ipv4.ip_forward = 0，Kubernetes需要启用" \
            "system_info"
    fi
    
    # 检查bridge-nf-call-iptables
    if pattern_exists "$system_info" "net.bridge.bridge-nf-call-iptables = 0"; then
        add_anomaly "$SEVERITY_WARNING" \
            "bridge-nf-call-iptables未启用" \
            "BRIDGE_NF_CALL_IPTABLES_DISABLED" \
            "net.bridge.bridge-nf-call-iptables = 0，Kubernetes需要启用" \
            "system_info"
    fi
    
    # 3. 检查ulimit限制
    if pattern_exists "$system_info" "open files.*1024"; then
        add_anomaly "$SEVERITY_INFO" \
            "文件句柄限制过低" \
            "LOW_ULIMIT_NOFILE" \
            "open files限制为1024，建议设置为65536或更高" \
            "system_info"
    fi
    
    # 4. 检查SELinux状态（可选）
    if pattern_exists "$system_info" "SELinux.*enforcing"; then
        add_anomaly "$SEVERITY_INFO" \
            "SELinux处于Enforcing模式" \
            "SELINUX_ENFORCING" \
            "SELinux处于Enforcing模式，可能影响Kubernetes运行" \
            "system_info"
    fi
}

# ============================================================================
# 结果输出
# ============================================================================

# 异常去重
deduplicate_anomalies() {
    if [[ ${#ANOMALIES[@]} -eq 0 ]]; then
        return
    fi
    
    local -a unique_anomalies=()
    local -A seen_anomalies
    
    for anomaly in "${ANOMALIES[@]}"; do
        IFS='|' read -r severity cn_name en_name details location <<< "$anomaly"
        
        # 使用英文标识符作为去重键
        if [[ -z "${seen_anomalies[$en_name]:-}" ]]; then
            seen_anomalies[$en_name]=1
            unique_anomalies+=("$anomaly")
        else
            log_debug "跳过重复异常: $en_name"
        fi
    done
    
    ANOMALIES=("${unique_anomalies[@]}")
}

# 排序异常（按严重级别）
sort_anomalies() {
    if [[ ${#ANOMALIES[@]} -eq 0 ]]; then
        return
    fi
    
    local -a critical=()
    local -a warning=()
    local -a info=()
    
    for anomaly in "${ANOMALIES[@]}"; do
        IFS='|' read -r severity _ _ _ _ <<< "$anomaly"
        case "$severity" in
            "$SEVERITY_CRITICAL")
                critical+=("$anomaly")
                ;;
            "$SEVERITY_WARNING")
                warning+=("$anomaly")
                ;;
            "$SEVERITY_INFO")
                info+=("$anomaly")
                ;;
        esac
    done
    
    ANOMALIES=()
    [[ ${#critical[@]} -gt 0 ]] && ANOMALIES+=("${critical[@]}")
    [[ ${#warning[@]} -gt 0 ]] && ANOMALIES+=("${warning[@]}")
    [[ ${#info[@]} -gt 0 ]] && ANOMALIES+=("${info[@]}")
}

# 输出文本格式报告
output_text_report() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local hostname=$(safe_cat "$DIAGNOSE_DIR/system_info" | grep -oP '(?<=hostname:).*' | head -1 | xargs || echo "未知")
    
    echo "=== Kubernetes节点诊断异常报告 ==="
    echo "诊断时间: $timestamp"
    echo "节点信息: $hostname"
    echo "分析目录: $DIAGNOSE_DIR"
    echo ""
    
    if [[ ${#ANOMALIES[@]} -eq 0 ]]; then
        echo -e "${GREEN}✓ 未检测到异常${NC}"
        echo ""
        echo "节点状态良好！"
        return
    fi
    
    local critical_count=0
    local warning_count=0
    local info_count=0
    
    # 统计各级别数量
    for anomaly in "${ANOMALIES[@]}"; do
        IFS='|' read -r severity _ _ _ _ <<< "$anomaly"
        case "$severity" in
            "$SEVERITY_CRITICAL") ((critical_count++)) || true ;;
            "$SEVERITY_WARNING") ((warning_count++)) || true ;;
            "$SEVERITY_INFO") ((info_count++)) || true ;;
        esac
    done
    
    # 输出严重级别异常
    if [[ $critical_count -gt 0 ]]; then
        echo "-------------------------------------------"
        echo "【严重级别】异常项"
        echo "-------------------------------------------"
        for anomaly in "${ANOMALIES[@]}"; do
            IFS='|' read -r severity cn_name en_name details location <<< "$anomaly"
            if [[ "$severity" == "$SEVERITY_CRITICAL" ]]; then
                echo -e "${RED}[严重]${NC} $cn_name | $en_name"
                echo "  详情: $details"
                echo "  位置: $location"
                echo ""
            fi
        done
    fi
    
    # 输出警告级别异常
    if [[ $warning_count -gt 0 ]]; then
        echo "-------------------------------------------"
        echo "【警告级别】异常项"
        echo "-------------------------------------------"
        for anomaly in "${ANOMALIES[@]}"; do
            IFS='|' read -r severity cn_name en_name details location <<< "$anomaly"
            if [[ "$severity" == "$SEVERITY_WARNING" ]]; then
                echo -e "${YELLOW}[警告]${NC} $cn_name | $en_name"
                echo "  详情: $details"
                echo "  位置: $location"
                echo ""
            fi
        done
    fi
    
    # 输出提示级别异常
    if [[ $info_count -gt 0 ]]; then
        echo "-------------------------------------------"
        echo "【提示级别】异常项"
        echo "-------------------------------------------"
        for anomaly in "${ANOMALIES[@]}"; do
            IFS='|' read -r severity cn_name en_name details location <<< "$anomaly"
            if [[ "$severity" == "$SEVERITY_INFO" ]]; then
                echo -e "${BLUE}[提示]${NC} $cn_name | $en_name"
                echo "  详情: $details"
                echo "  位置: $location"
                echo ""
            fi
        done
    fi
    
    # 输出统计
    echo "-------------------------------------------"
    echo "异常统计"
    echo "-------------------------------------------"
    echo "严重: $critical_count 项"
    echo "警告: $warning_count 项"
    echo "提示: $info_count 项"
    echo "总计: ${#ANOMALIES[@]} 项"
}

# 输出JSON格式报告
output_json_report() {
    local timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    local hostname=$(safe_cat "$DIAGNOSE_DIR/system_info" | grep -oP '(?<=hostname:).*' | head -1 | xargs || echo "unknown")
    
    echo "{"
    echo "  \"report_version\": \"1.0\","
    echo "  \"timestamp\": \"$timestamp\","
    echo "  \"hostname\": \"$hostname\","
    echo "  \"diagnose_dir\": \"$DIAGNOSE_DIR\","
    echo "  \"anomalies\": ["
    
    local first=true
    for anomaly in "${ANOMALIES[@]}"; do
        IFS='|' read -r severity cn_name en_name details location <<< "$anomaly"
        
        if [[ "$first" == true ]]; then
            first=false
        else
            echo ","
        fi
        
        echo -n "    {"
        echo -n "\"severity\": \"$severity\", "
        echo -n "\"cn_name\": \"$cn_name\", "
        echo -n "\"en_name\": \"$en_name\", "
        echo -n "\"details\": \"$details\", "
        echo -n "\"location\": \"$location\""
        echo -n "}"
    done
    
    echo ""
    echo "  ],"
    echo "  \"summary\": {"
    
    local critical_count=0
    local warning_count=0
    local info_count=0
    
    if [[ ${#ANOMALIES[@]} -gt 0 ]]; then
        for anomaly in "${ANOMALIES[@]}"; do
            IFS='|' read -r severity _ _ _ _ <<< "$anomaly"
            case "$severity" in
                "$SEVERITY_CRITICAL") ((critical_count++)) || true ;;
                "$SEVERITY_WARNING") ((warning_count++)) || true ;;
                "$SEVERITY_INFO") ((info_count++)) || true ;;
            esac
        done
    fi
    
    echo "    \"critical\": $critical_count,"
    echo "    \"warning\": $warning_count,"
    echo "    \"info\": $info_count,"
    echo "    \"total\": ${#ANOMALIES[@]}"
    echo "  }"
    echo "}"
}

# 生成并输出报告
generate_report() {
    log_info "生成报告..."
    
    # 去重和排序
    deduplicate_anomalies
    sort_anomalies
    
    # 根据格式输出
    local output_content
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        output_content=$(output_json_report)
    else
        output_content=$(output_text_report)
    fi
    
    # 输出到控制台或文件
    if [[ -n "$OUTPUT_FILE" ]]; then
        echo "$output_content" > "$OUTPUT_FILE"
        log_info "报告已保存到: $OUTPUT_FILE"
        # 同时输出到控制台（简化版）
        if [[ "$OUTPUT_FORMAT" != "json" ]]; then
            echo "$output_content"
        fi
    else
        echo "$output_content"
    fi
}

# ============================================================================
# 主函数
# ============================================================================

main() {
    parse_arguments "$@"
    
    log_info "kudig.sh v$VERSION - Kubernetes节点诊断分析工具"
    
    # 环境检查
    check_required_commands
    validate_diagnose_dir "$DIAGNOSE_DIR"
    
    # 执行所有检测器
    check_system_resources
    check_process_services
    check_network
    check_kernel
    check_container_runtime
    check_kubernetes
    check_time_sync
    check_configuration
    
    # 生成报告
    generate_report
    
    # 根据异常数量返回退出码
    local critical_count=0
    if [[ ${#ANOMALIES[@]} -gt 0 ]]; then
        for anomaly in "${ANOMALIES[@]}"; do
            IFS='|' read -r severity _ _ _ _ <<< "$anomaly"
            if [[ "$severity" == "$SEVERITY_CRITICAL" ]]; then
                ((critical_count++)) || true
            fi
        done
    fi
    
    if [[ $critical_count -gt 0 ]]; then
        exit 2  # 有严重异常
    elif [[ ${#ANOMALIES[@]} -gt 0 ]]; then
        exit 1  # 有警告或提示
    else
        exit 0  # 无异常
    fi
}

# 执行主函数
main "$@"
