# kudig.sh - Kubernetes节点诊断日志分析工具

## 简介

`kudig.sh` 是一个用于分析Kubernetes节点诊断日志的Shell脚本工具。它能够自动分析 `diagnose_k8s.sh` 收集的诊断数据，识别各类异常情况，并生成中英文对照的诊断报告。

## 功能特性

- ✅ **全面的异常检测**：涵盖系统资源、进程服务、网络、内核、容器运行时、Kubernetes组件等多个维度
- ✅ **双语输出**：同时提供中文异常描述和英文异常标识符
- ✅ **严重级别分类**：异常按严重、警告、提示三个级别分类
- ✅ **多种输出格式**：支持文本和JSON格式输出
- ✅ **本地化分析**：完全在本地执行，无需外部依赖
- ✅ **智能去重**：自动去除重复的异常项

## 安装

将脚本下载到本地并添加执行权限：

```bash
chmod +x kudig.sh
```

## 使用方法

### 基本用法

```bash
./kudig.sh <诊断目录>
```

示例：
```bash
./kudig.sh /tmp/diagnose_1702468800
```

### 命令行选项

| 选项 | 说明 |
|-----|------|
| `-h, --help` | 显示帮助信息 |
| `-v, --version` | 显示版本信息 |
| `--verbose` | 详细输出模式，显示调试信息 |
| `--json` | 输出JSON格式报告 |
| `-o, --output <文件>` | 保存报告到指定文件 |

### 使用示例

1. **基本分析**：
```bash
./kudig.sh /tmp/diagnose_1702468800
```

2. **详细模式**：
```bash
./kudig.sh --verbose /tmp/diagnose_1702468800
```

3. **输出JSON格式**：
```bash
./kudig.sh --json /tmp/diagnose_1702468800 > report.json
```

4. **保存报告到文件**：
```bash
./kudig.sh -o report.txt /tmp/diagnose_1702468800
```

## 输出示例

### 文本格式输出

```
=== Kubernetes节点诊断异常报告 ===
诊断时间: 2024-12-13 10:30:00
节点信息: k8s-node-01
分析目录: /tmp/diagnose_1702468800

-------------------------------------------
【严重级别】异常项
-------------------------------------------
[严重] 系统负载过高 | HIGH_SYSTEM_LOAD
  详情: 15分钟平均负载 18.5，超过CPU核心数(4)的4倍
  位置: system_status
  
[严重] Kubelet服务未运行 | KUBELET_SERVICE_DOWN
  详情: kubelet.service状态为failed
  位置: daemon_status/kubelet_status

-------------------------------------------
【警告级别】异常项
-------------------------------------------
[警告] 连接跟踪表使用率高 | CONNTRACK_TABLE_HIGH_USAGE
  详情: 当前连接数 45678/65536 (70%)
  位置: network_info

-------------------------------------------
【提示级别】异常项
-------------------------------------------
[提示] Swap未禁用 | SWAP_NOT_DISABLED
  详情: Kubernetes节点建议禁用swap，当前 2048KB
  位置: system_info

-------------------------------------------
异常统计
-------------------------------------------
严重: 2项
警告: 1项
提示: 1项
总计: 4项
```

### JSON格式输出

```json
{
  "report_version": "1.0",
  "timestamp": "2024-12-13T02:30:00Z",
  "hostname": "k8s-node-01",
  "diagnose_dir": "/tmp/diagnose_1702468800",
  "anomalies": [
    {
      "severity": "严重",
      "cn_name": "系统负载过高",
      "en_name": "HIGH_SYSTEM_LOAD",
      "details": "15分钟平均负载 18.5，超过CPU核心数(4)的4倍",
      "location": "system_status"
    }
  ],
  "summary": {
    "critical": 2,
    "warning": 1,
    "info": 1,
    "total": 4
  }
}
```

## 异常检测规则

### 系统资源类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| 系统负载过高 | HIGH_SYSTEM_LOAD | 严重 | 15分钟负载超过CPU核心数的4倍 |
| 系统负载偏高 | ELEVATED_SYSTEM_LOAD | 警告 | 15分钟负载超过CPU核心数的2倍 |
| 内存使用率过高 | HIGH_MEMORY_USAGE | 严重 | 内存使用率≥95% |
| 内存使用率偏高 | ELEVATED_MEMORY_USAGE | 警告 | 内存使用率≥85% |
| 磁盘空间严重不足 | DISK_SPACE_CRITICAL | 严重 | 磁盘使用率≥95% |
| 磁盘空间不足 | DISK_SPACE_LOW | 警告 | 磁盘使用率≥90% |
| 文件句柄使用量过高 | HIGH_FILE_HANDLES | 警告 | 进程文件句柄数>50000 |
| 进程/线程数异常 | PID_LEAK_DETECTED | 严重 | 进程线程数>10000 |
| Inode使用率过高 | HIGH_INODE_USAGE | 警告 | Inode使用率≥90% |

### 进程与服务类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| Kubelet服务未运行 | KUBELET_SERVICE_DOWN | 严重 | kubelet服务状态为failed |
| 容器运行时服务异常 | CONTAINER_RUNTIME_DOWN | 严重 | docker和containerd均为failed |
| ps命令挂起 | PS_COMMAND_HUNG | 严重 | ps命令执行挂起 |
| 存在D状态进程 | PROCESS_IN_D_STATE | 严重 | 检测到不可中断睡眠状态进程 |
| runc进程可能挂起 | RUNC_PROCESS_HANG | 警告 | runc进程可能处于挂起状态 |
| Firewalld服务运行中 | FIREWALLD_RUNNING | 警告 | K8s节点建议关闭firewalld |

### 网络类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| 连接跟踪表满 | CONNTRACK_TABLE_FULL | 严重 | 连接跟踪表使用率≥95% |
| 连接跟踪表使用率高 | CONNTRACK_TABLE_HIGH_USAGE | 警告 | 连接跟踪表使用率≥80% |
| 网卡接口down | NETWORK_INTERFACE_DOWN | 警告 | 网卡处于down状态 |
| 缺少默认路由 | NO_DEFAULT_ROUTE | 警告 | 未检测到默认路由 |
| Kubelet端口未监听 | KUBELET_PORT_NOT_LISTENING | 严重 | 10250端口未监听 |
| iptables规则过多 | TOO_MANY_IPTABLES_RULES | 警告 | iptables规则数>50000 |

### 内核与驱动类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| 内核Panic | KERNEL_PANIC | 严重 | 内核发生panic事件 |
| 内核触发OOM杀进程 | KERNEL_OOM_KILLER | 严重 | 内核OOM Killer被触发 |
| 系统内存不足 | SYSTEM_OUT_OF_MEMORY | 严重 | 系统日志显示内存不足 |
| 文件系统只读 | FILESYSTEM_READONLY | 严重 | 文件系统被重新挂载为只读 |
| 磁盘IO错误 | DISK_IO_ERROR | 严重 | 检测到多次IO错误 |
| 内核模块加载失败 | KERNEL_MODULE_LOAD_FAILED | 警告 | 内核模块加载失败 |

### 容器运行时类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| Docker启动失败 | DOCKER_START_FAILED | 严重 | Docker服务启动失败 |
| Docker存储驱动错误 | DOCKER_STORAGE_DRIVER_ERROR | 严重 | Docker存储驱动出现错误 |
| 容器创建失败率高 | CONTAINER_CREATE_FAILED | 警告 | 容器创建失败次数过多 |
| 镜像拉取失败 | IMAGE_PULL_FAILED | 警告 | 镜像拉取失败次数过多 |

### Kubernetes组件类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| Kubelet PLEG不健康 | KUBELET_PLEG_UNHEALTHY | 严重 | Pod生命周期事件生成器不健康 |
| CNI网络插件错误 | CNI_PLUGIN_ERROR | 严重 | CNI网络插件失败 |
| 证书已过期 | CERTIFICATE_EXPIRED | 严重 | Kubelet证书已过期 |
| 证书即将过期 | CERTIFICATE_EXPIRING | 警告 | Kubelet证书即将过期 |
| API Server连接失败 | APISERVER_CONNECTION_FAILED | 严重 | 无法连接到API Server |
| Kubelet认证失败 | KUBELET_AUTH_FAILED | 严重 | Kubelet认证失败 |
| Pod被驱逐 | POD_EVICTED | 警告 | Pod被驱逐，可能资源不足 |
| 节点NotReady状态 | NODE_NOT_READY | 严重 | 节点处于NotReady状态 |
| 磁盘压力 | DISK_PRESSURE | 警告 | 节点存在磁盘压力 |
| 内存压力 | MEMORY_PRESSURE | 警告 | 节点存在内存压力 |

### 配置类

| 中文名称 | 英文标识符 | 严重级别 | 说明 |
|---------|-----------|---------|------|
| 时间同步服务未运行 | TIME_SYNC_SERVICE_DOWN | 提示 | ntpd和chronyd均未运行 |
| Swap未禁用 | SWAP_NOT_DISABLED | 提示 | K8s节点建议禁用swap |
| IP转发未启用 | IP_FORWARD_DISABLED | 警告 | net.ipv4.ip_forward=0 |
| bridge-nf-call-iptables未启用 | BRIDGE_NF_CALL_IPTABLES_DISABLED | 警告 | 内核参数需要启用 |
| 文件句柄限制过低 | LOW_ULIMIT_NOFILE | 提示 | ulimit open files建议≥65536 |
| SELinux处于Enforcing模式 | SELINUX_ENFORCING | 提示 | SELinux可能影响K8s运行 |

## 退出码

| 退出码 | 说明 |
|-------|------|
| 0 | 未检测到异常 |
| 1 | 检测到警告或提示级别异常 |
| 2 | 检测到严重级别异常 |

## 系统要求

- **操作系统**：Linux（Red Hat、CentOS、Aliyun Linux、Kylin等）
- **Shell**：bash 4.0+
- **必需命令**：grep, awk, sed, wc, sort, uniq, tail, head, find

## 工作流程

1. **数据收集**：使用 `diagnose_k8s.sh` 收集节点诊断数据
2. **数据分析**：运行 `kudig.sh` 分析诊断数据
3. **异常识别**：自动检测各类异常情况
4. **报告生成**：生成中英文对照的诊断报告

```
┌─────────────────┐
│ diagnose_k8s.sh │ ──► 收集诊断数据
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   诊断目录       │
│  /tmp/diagnose_ │
│   ${timestamp}  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   kudig.sh      │ ──► 分析诊断数据
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   诊断报告       │
│ (文本/JSON)     │
└─────────────────┘
```

## 典型使用场景

### 场景1：节点故障快速诊断

```bash
# 1. 收集诊断数据
sudo ./diagnose_k8s.sh

# 2. 分析诊断数据
./kudig.sh /tmp/diagnose_1702468800

# 3. 根据报告中的异常标识符查找解决方案
```

### 场景2：自动化巡检

```bash
#!/bin/bash
# 定期巡检脚本

# 收集诊断数据
sudo /opt/scripts/diagnose_k8s.sh

# 分析最新的诊断数据
LATEST_DIAGNOSE=$(ls -t /tmp/diagnose_* | head -1)
/opt/scripts/kudig.sh --json "$LATEST_DIAGNOSE" > /var/log/kudig_report.json

# 检查退出码
if [ $? -eq 2 ]; then
    # 发现严重异常，发送告警
    send_alert "严重异常" /var/log/kudig_report.json
fi
```

### 场景3：与监控系统集成

```bash
# 生成JSON格式报告
./kudig.sh --json /tmp/diagnose_1702468800 | \
    curl -X POST -H "Content-Type: application/json" \
    -d @- http://monitoring-server/api/diagnostics
```

## 故障排除

### 问题1：脚本提示命令不存在

**解决方案**：安装缺失的命令
```bash
# CentOS/RHEL
yum install -y grep gawk sed coreutils findutils

# Ubuntu/Debian
apt-get install -y grep gawk sed coreutils findutils
```

### 问题2：诊断目录结构不完整

**现象**：警告信息"诊断目录结构可能不完整"

**解决方案**：确保使用完整的 `diagnose_k8s.sh` 脚本收集数据，并以root权限执行

### 问题3：无法读取某些日志文件

**现象**：某些检测项没有结果

**解决方案**：
- 确保诊断数据收集时有足够权限
- 检查日志文件是否存在于诊断目录中

## 版本历史

- **v1.0.0** (2024-12-13)
  - 初始版本发布
  - 支持8大类异常检测
  - 支持文本和JSON输出格式

## 贡献

欢迎提交Issue和Pull Request来改进此工具。

## 许可证

本项目采用 Apache License 2.0 许可证。

## 联系方式

如有问题或建议，请通过以下方式联系：
- 提交GitHub Issue
- 发送邮件至项目维护者

---

**注意**：本工具仅用于诊断分析，不会修改任何系统配置或日志文件。
