# kudig — 已归并至 klaw

> ⚠️ **本仓库所有功能已完整迁移至 [klaw](../klaw)，源代码已全部清除。**
>
> | kudig 组件 | klaw 目标位置 |
> |-----------|--------------|
> | 诊断核心 (20 pkg: types/analyzer/collector/rca/autofix/reporter/rules/cost/scanner/ai/history/notifier/tui/ebpf 等) | `klaw/internal/diag/` |
> | CLI 入口 (cmd/) | `klaw/cmd/klaw/` (`klaw diag` 子命令) |
> | K8s Operator (CRD + 控制器) | `klaw/operator/` |
> | etcd 备份模块 | `klaw/modules/etcd-backup/` |
> | v1-bash 诊断脚本 | 功能已被 Go 分析器取代 |
>
> klaw 现为唯一活跃仓库。所有功能可通过 `klaw diag` CLI 或 `GET /api/v1/diag/run` API 使用。

> **快速选择**: 
> - ✅ [v1.0 Bash 版本](v1-bash/) - **生产可用**，轻量级 Bash 脚本
> - ✅ [v2.0 Go 版本](v2-go/) - **生产可用**，功能完整，70+ 分析器

## 项目简介

`kudig` 是一个强大的 Kubernetes 节点诊断工具，能够自动识别各类异常情况并生成中英文对照的诊断报告。

## 版本说明

### v1.2.0 - Bash 版本 ✅ 生产可用

**位置**: [`v1-bash/`](v1-bash/)

- **实现**: Bash 脚本
- **状态**: 稳定可用
- **特性**: 120+项异常检测规则，离线分析模式，生产环境就绪
- **优势**: 轻量级、无依赖、开箱即用
- **文档**: [v1-bash/README.md](v1-bash/README.md)

**快速使用**:
```bash
cd v1-bash
./kudig /tmp/diagnose_1702468800
```

### v2.0 - Go 版本 ✅ 生产可用

**位置**: [`v2-go/`](v2-go/)

- **实现**: Go 语言
- **状态**: 生产阶段，功能完整（100%）
- **特性**: 
  - 双模式支持（离线 + 在线）
  - **70+ 内置分析器**（9 大类：系统、进程、网络、内核、K8s、运行时、安全、eBPF、服务网格）
  - 多输出格式（Text/JSON/HTML/**SARIF**）
  - YAML 规则引擎
  - Kubernetes 原生部署
  - Docker 镜像
- **文档**: [v2-go/README.md](v2-go/README.md)

**新增亮点**（Phase 4-8）功能特性 **100% 完成**：

#### Phase 4-7（已完成）
- DNS 诊断: CoreDNS 健康检查、Pod DNS 配置分析
- 存储分析: PVC/PV 状态检查、CSI Driver 监控
- GPU/NPU 诊断: NVIDIA GPU、华为 Ascend NPU 支持
- CIS 安全扫描: Kubernetes CIS Benchmark 合规检查
- RBAC 审计: 权限分析、危险权限检测
- Operator 模式: Kubernetes CRD + 控制器，支持定时诊断
- eBPF 深度诊断: TCP/DNS/文件 I/O 实时追踪（内核 4.18+）
- AI/LLM 辅助: OpenAI/Qwen/Ollama 智能分析与修复建议

#### Phase 8（新增，功能特性 100%）
- TUI 交互模式: bubbletea 终端 UI，直观菜单驱动
- 服务网格诊断: Istio/Linkerd 控制平面、sidecar 状态检查
- 根因分析 (RCA): 多症状智能关联，定位根本问题
- 自动修复引擎: 安全、低风险的自动修复
- SARIF 安全报告: GitHub/CodeQL 兼容的安全扫描格式
- Grafana Dashboard: 官方 Dashboard JSON 导出
- 镜像安全扫描: Trivy 集成，CVE 漏洞检测
- 成本分析: Kubernetes 资源成本估算与优化建议
- 性能剖析: pprof 支持，CPU/内存/Goroutine 分析
- 分布式追踪: OpenTelemetry 支持
- 多集群联邦诊断: 跨集群统一诊断视图
- Shell 自动补全: Bash/Zsh/Fish 全支持

**快速使用**:
```bash
cd v2-go
make build
./build/kudig offline /tmp/diagnose_1702468800
./build/kudig online --all-nodes
./build/kudig tui  # 交互式 TUI 模式
```

## 项目结构

```
kudig/
├── v1-bash/              # v1.0 Bash 版本（生产可用）
│   ├── kudig             # 主脚本
│   ├── README.md         # v1.0 文档
│   ├── TESTING.md        # 测试说明
│   └── reference/        # 示例诊断数据
│
├── v2-go/                # v2.0 Go 版本（生产可用，功能 100%）
│   ├── cmd/              # CLI 入口
│   ├── pkg/              # 核心包
│   │   ├── analyzer/     # 分析器（70+）
│   │   ├── collector/    # 数据收集层
│   │   ├── reporter/     # 报告生成层（Text/JSON/HTML/SARIF）
│   │   ├── rules/        # 规则引擎
│   │   ├── rca/          # 根因分析
│   │   ├── autofix/      # 自动修复
│   │   ├── cost/         # 成本分析
│   │   ├── scanner/      # 镜像扫描
│   │   └── tui/          # TUI 界面
│   ├── charts/           # Helm Chart
│   ├── operator/         # Kubernetes Operator
│   ├── Makefile          # 构建脚本
│   ├── Dockerfile        # Docker 构建
│   └── README.md         # v2.0 文档
│
├── docs/                 # 文档目录
│   ├── CNCF_GRADUATION_GAP_ANALYSIS.md
│   └── FUNCTIONAL_ROADMAP.md
├── scripts/              # 辅助脚本
├── tests/                # 测试文件
├── README.md             # 本文档
├── ROADMAP.md            # 项目路线图
├── STRUCTURE.md          # 项目结构说明
└── LICENSE               # 许可证
```

## 快速开始

### 使用 v1.0 Bash 版本（推荐）

1. **进入 v1-bash 目录**
```bash
cd v1-bash
```

2. **收集诊断数据**
```bash
sudo ./diagnose_k8s.sh
# 生成目录: /tmp/diagnose_1702468800
```

3. **分析诊断数据**
```bash
./kudig /tmp/diagnose_1702468800
```

详细使用说明请查看 [v1-bash/README.md](v1-bash/README.md)

### 使用 v2.0 Go 版本

```bash
cd v2-go
make deps
make build

# 离线分析
./build/kudig offline /tmp/diagnose_1702468800

# 在线诊断
./build/kudig online --all-nodes

# 交互式 TUI 模式
./build/kudig tui

# 根因分析
./build/kudig rca

# 成本分析
./build/kudig cost

# 镜像安全扫描
./build/kudig scan nginx:latest

# 导出 Grafana Dashboard
./build/kudig grafana > dashboard.json

# Shell 自动补全
source <(./build/kudig completion bash)
```

详细开发说明请查看 [v2-go/README.md](v2-go/README.md)

## 核心特性对比

| 特性 | v1.0 Bash | v2.0 Go |
|-----|----------|--------|
| **状态** | 生产可用 | 生产可用 |
| **实现语言** | Bash | Go |
| **离线分析** | 支持 | 支持 |
| **在线诊断** | 不支持 | 支持 |
| **分析器数量** | 80+ 规则 | 70+ 分析器 |
| **输出格式** | Text/JSON | Text/JSON/HTML/SARIF |
| **TUI 界面** | 不支持 | 支持 |
| **根因分析** | 不支持 | 支持 |
| **自动修复** | 不支持 | 支持 |
| **成本分析** | 不支持 | 支持 |
| **镜像扫描** | 不支持 | 支持 |
| **排查建议** | 支持 | 支持 |
| **自定义规则** | 不支持 | 支持 YAML规则 |
| **K8s部署** | 不支持 | 支持 Helm Chart/Operator |
| **eBPF 诊断** | 不支持 | 支持 |
| **服务网格** | 不支持 | 支持 |
| **Shell 补全** | 不支持 | 支持 |

## 内置检测规则（80+项）

kudig.sh 内置了多个类别的80+项检测规则，自动识别各类异常情况。

### 1. 系统资源检测（6项）
- **CPU负载检测**：检查15分钟平均负载是否超过CPU核心数2倍/4倍
- **内存使用检测**：检测内存使用率是否超过85%/95%
- **磁盘空间检测**：检测挂载点使用率是否超过90%/95%
- **文件句柄检测**：检测进程文件句柄数是否过高(>50000)
- **进程/线程数检测**：检测PID泄漏，某进程线程数>5000/10000
- **磁盘使用检测**：检测磁盘使用率是否超过90%

### 2. 进程与服务检测（6项）
- **Kubelet服务检测**：检测kubelet服务状态（running/failed/stopped）
- **容器运行时检测**：检测docker/containerd服务状态
- **ps命令检测**：检测ps命令是否挂起
- **D状态进程检测**：检测是否存在D状态（不可中断睡眠）进程
- **Runc进程检测**：检测runc进程数是否异常(>1000)
- **Firewalld检测**：检测Firewalld是否在运行（可能影响K8s网络）

### 3. 网络检测（5项）
- **连接追踪表检测**：检测conntrack表使用率>80%/95%
- **网卡状态检测**：检测网卡是否DOWN（排除lo/veth）
- **默认路由检测**：检测是否配置默认路由
- **端口监听检测**：检测kubelet端口(10250)是否在监听
- **Iptables规则检测**：检测iptables规则数是否过多(>50000)

### 4. 内核检测（7项）
- **内核Panic检测**：检测dmesg中是否有kernel panic
- **OOM Killer检测**：检测是否发生OOM事件
- **Messages日志OOM检测**：检测/var/log/messages中的OOM事件
- **文件系统错误检测**：检测文件系统是否变为只读
- **磁盘IO错误检测**：检测磁盘IO错误次数
- **内核模块检测**：检测是否有内核模块加载失败
- **NMI Watchdog检测**：检测NMI watchdog事件

### 5. 容器运行时检测（4项）
- **Docker启动检测**：检测docker启动错误
- **Docker存储驱动检测**：检测存储驱动错误
- **Containerd容器创建检测**：检测容器创建失败次数
- **镜像拉取检测**：检测镜像拉取失败错误

### 6. Kubernetes组件检测（9项）
- **PLEG状态检测**：检测PLEG is not healthy错误
- **CNI插件检测**：检测CNI插件错误（network plugin not ready）
- **Kubelet证书检测**：检测Kubernetes证书是否过期
- **API Server连接检测**：检测与API Server的连接问题
- **Kubelet认证检测**：检测认证失败错误
- **Pod驱逐检测**：检测Pod驱逐事件
- **节点状态检测**：检测节点是否Ready
- **磁盘压力检测**：检测节点磁盘压力
- **内存压力检测**：检测节点内存压力

### 7. 时间同步检测（1项）
- **NTP/Chrony状态检测**：检测时间同步服务状态（ntpd/chronyd）

### 8. 配置检测（5项）
- **Swap配置检测**：检测Swap是否开启（K8s不建议开启）
- **IP转发检测**：检测net.ipv4.ip_forward是否启用
- **bridge-nf-call-iptables检测**：检测桥接流量是否经过iptables
- **ulimit检测**：检测open files限制是否过低
- **SELinux检测**：检测SELinux是否为Enforcing

### 9. 存储和文件系统检测（4项）
- **磁盘IO等待检测**：检测磁盘IO等待时间是否过高(>20%/50%)
- **文件系统挂载选项检测**：检测是否使用noatime挂载选项
- **文件系统错误检测**：检测EXT4文件系统错误
- **NFS挂载状态检测**：检测NFS连接问题

### 10. 安全配置检测（4项）
- **密码策略检测**：检测密码过期策略是否配置
- **SSH配置检测**：检测是否允许root直接登录
- **防火墙状态检测**：检测防火墙是否运行
- **AppArmor/SELinux检测**：检测安全模块状态

### 11. 性能指标检测（4项）
- **上下文切换率检测**：检测每秒上下文切换次数(>100000)
- **系统调用率检测**：检测每秒系统调用次数(>500000)
- **缓存使用检测**：检测缓存使用率是否过高(>80%)
- **中断率检测**：检测每秒中断次数(>50000)

### 12. 日志深度分析检测（5项）
- **Kubelet错误统计**：检测Kubelet日志中错误数量(>100)
- **容器重启检测**：检测容器频繁重启(>10次)
- **镜像拉取超时检测**：检测镜像拉取超时(>5次)
- **存储空间检测**：检测存储空间不足错误
- **内核错误统计**：检测系统日志中内核错误(>50)

### 13. 容器和Pod状态检测（5项）
- **容器创建失败检测**：检测容器创建失败次数(>10)
- **容器启动超时检测**：检测容器启动超时(>5次)
- **Pod挂载失败检测**：检测Pod挂载失败(>5次)
- **Pod沙箱创建失败检测**：检测Pod沙箱创建失败(>5次)
- **容器运行时健康检测**：检测容器运行时健康状态

### 14. 网络性能检测（5项）
- **网络错误检测**：检测网络接收/发送错误(>100)
- **网络丢包检测**：检测网络接收/发送丢包(>1000)
- **TCP连接状态检测**：检测TIME_WAIT连接过多(>10000)
- **网络带宽检测**：检测网络带宽使用率
- **DNS配置检测**：检测DNS服务器配置

## 输出示例

### 文本格式（默认）

**正常情况（有少量警告）：**

```
================================================================
  kudig v1.2.0 - Kubernetes节点诊断分析工具
================================================================

诊断目录: /tmp/diagnose_1702468800
分析时间: 2026-01-16 10:50:01

开始诊断检查...

========== 系统资源检查 ==========
  [OK] CPU负载: 正常 (15min负载: 0.34, CPU核心: 8)
  [OK] 内存使用: 正常 (使用率: 32%)
  [OK] 磁盘空间: 正常 (所有挂载点使用率<90%)
  [OK] 文件句柄: 正常 (最大: 273)
  [OK] 进程/线程数: 正常 (最大线程数: 33)
  [OK] 磁盘使用: 正常 (所有挂载点<90%)

========== 进程与服务检查 ==========
  [OK] Kubelet服务: running
  [OK] 容器运行时: docker=unknown, containerd=running
  [OK] ps命令: 正常
  [OK] D状态进程: 未发现
  [OK] runc进程: 正常
  [OK] Firewalld: 已关闭

========== 网络状态检查 ==========
  [OK] 连接跟踪表: 正常 (517/262144, 0%)
  [!] 网卡状态: 部分down (kube-ipvs0,nodelocaldns)
  [OK] 默认路由: 已配置
  [OK] Kubelet端口(10250): 正常监听
  [OK] iptables规则: 正常 (44 条)

========== 内核状态检查 ==========
  [OK] 内核Panic: 未发现
  [OK] OOM Killer: 未触发
  [OK] messages日志OOM: 未发现
  [OK] 文件系统: 正常
  [OK] 磁盘IO: 正常 (0 次错误)
  [!] 内核模块: 存在加载失败
  [!] NMI Watchdog: 被触发

========== 容器运行时检查 ==========
  [OK] Docker启动: 正常
  [OK] Docker存储驱动: 正常
  [OK] Containerd容器创建: 正常 (0 次失败)
  [OK] 镜像拉取: 正常 (0 次失败)

========== Kubernetes组件检查 ==========
  [OK] PLEG状态: 健康
  [OK] CNI网络插件: 正常
  [OK] Kubelet证书: 正常
  [OK] API Server连接: 正常 (0 次失败)
  [OK] Kubelet认证: 正常
  [OK] Pod驱逐: 未发现
  [OK] 节点状态: Ready
  [OK] 磁盘压力: 无
  [OK] 内存压力: 无

========== 时间同步检查 ==========
  [!] 时间同步: ntpd=unknown, chronyd=unknown (建议启用)

========== 系统配置检查 ==========
  [OK] Swap配置: 已禁用
  [OK] IP转发: 已启用
  [OK] bridge-nf-call-iptables: 已启用
  [OK] ulimit open files: 正常
  [OK] SELinux: 非Enforcing

================================================================
  诊断结果汇总
================================================================
=== Kubernetes节点诊断异常报告 ===
诊断时间: 2026-01-16 10:50:11
节点信息: k8s-node-01
分析目录: /tmp/diagnose_1702468800

-------------------------------------------
【警告级别】异常项
-------------------------------------------
[警告] 网卡接口down | NETWORK_INTERFACE_DOWN
  详情: 以下网卡处于down状态: kube-ipvs0,nodelocaldns
  位置: network_info

[警告] 内核模块加载失败 | KERNEL_MODULE_LOAD_FAILED
  详情: 存在内核模块加载失败
  位置: logs/dmesg.log

-------------------------------------------
【提示级别】异常项
-------------------------------------------
[提示] 时间同步服务未运行 | TIME_SYNC_SERVICE_DOWN
  详情: ntpd和chronyd服务均未运行
  位置: service_status

-------------------------------------------
异常统计
-------------------------------------------
严重: 0 项
警告: 2 项
提示: 1 项
总计: 3 项
```

**有严重异常情况：**

```
========== 进程与服务检查 ==========
  [FAIL] Kubelet服务: failed
    -> 建议: 检查kubelet日志: journalctl -u kubelet -n 100; systemctl restart kubelet

========== 系统资源检查 ==========
  [FAIL] 磁盘空间 [/]: 严重不足 (使用率: 96%)
    -> 建议: 检查占用空间大的目录: du -sh /* | sort -rh | head

-------------------------------------------
【严重级别】异常项
-------------------------------------------
[严重] Kubelet服务未运行 | KUBELET_SERVICE_DOWN
  详情: kubelet.service状态为failed
  位置: daemon_status/kubelet_status

[严重] 磁盘空间严重不足 | DISK_SPACE_CRITICAL
  详情: 挂载点 / 使用率 96%
  位置: system_status
```

### JSON格式

```json
{
  "metadata": {
    "report_version": "1.0",
    "timestamp": "2024-12-13T13:47:00Z",
    "hostname": "k8s-node-01",
    "diagnose_dir": "/tmp/diagnose_1702468800"
  },
  "anomalies": [
    {
      "severity": "严重",
      "cn_name": "Kubelet服务未运行",
      "en_name": "KUBELET_SERVICE_DOWN",
      "details": "kubelet.service状态为failed",
      "location": "daemon_status/kubelet_status"
    },
    {
      "severity": "警告",
      "cn_name": "磁盘空间不足",
      "en_name": "DISK_SPACE_LOW",
      "details": "挂载点 / 使用率 92%",
      "location": "system_status"
    }
  ],
  "summary": {
    "critical": 1,
    "warning": 1,
    "info": 0,
    "total": 2
  }
}
```

## 项目状态与路线图

### 当前状态评估

| 维度 | v1-Bash | v2-Go | 状态说明 |
|------|---------|-------|----------|
| 功能完整性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 两个版本均生产可用，v2.0 功能特性 100% |
| 测试覆盖 | ⭐⭐⭐☆☆ | ⭐⭐⭐⭐☆ | 核心包测试覆盖 60%+，目标 80%+ |
| 代码质量 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐☆ | golangci-lint 已配置 |
| 安全合规 | ⭐⭐⭐⭐☆ | ⭐⭐⭐⭐☆ | SLSA Level 1 达成，Level 2 进行中 |
| 文档完善 | ⭐⭐⭐⭐☆ | ⭐⭐⭐⭐⭐ | 文档完整，包含所有新功能 |
| 可观测性 | ⭐⭐☆☆☆ | ⭐⭐⭐⭐☆ | Prometheus metrics、Grafana Dashboard |
| 社区建设 | ⭐⭐☆☆☆ | ⭐⭐☆☆☆ | 开源规范待完善 |

### 🔥 当前重点（P0 优先级）

我们正在执行 [ROADMAP.md](./ROADMAP.md) 中的改进计划：

**本月目标**：
- [x] 编译修复：删除重复文件，修复编译错误
- [x] 测试修复：修复 notifier/history 测试失败
- [x] 测试补全：所有 pkg 包均有测试文件
- [ ] 测试覆盖率：v2-go 单元测试覆盖率 80%+
- [ ] SLSA Level 2：cosign 签名容器镜像
- [ ] OpenSSF Badge：获取 Best Practices Passing 徽章
- [ ] 代码质量：golangci-lint 全规则通过

### 完整路线图

查看详细的改进计划和待办清单：
📋 **[ROADMAP.md](./ROADMAP.md)** - 基于 ISO/IEC 25010、CNCF、SLSA 等行业标准的完整规划

查看 CNCF 毕业差距分析：
📋 **[docs/CNCF_GRADUATION_GAP_ANALYSIS.md](./docs/CNCF_GRADUATION_GAP_ANALYSIS.md)**

---

## 文档索引

- **主文档**: [README.md](README.md) - 本文档
- **v1.0 Bash**: [v1-bash/README.md](v1-bash/README.md) - 生产可用版本
  - [v1-bash/TESTING.md](v1-bash/TESTING.md) - 测试说明
- **v2.0 Go**: [v2-go/README.md](v2-go/README.md) - 生产可用版本
- **路线图**: [ROADMAP.md](./ROADMAP.md) - 项目改进计划
- **结构说明**: [STRUCTURE.md](./STRUCTURE.md) - 项目结构详细说明
- **质量报告**: [TEST_REPORT.md](TEST_REPORT.md) - 代码质量检查
- **CNCF 分析**: [docs/CNCF_GRADUATION_GAP_ANALYSIS.md](docs/CNCF_GRADUATION_GAP_ANALYSIS.md)

## 版本选择指南

### 何时使用 v1.0 Bash 版本？

✅ **推荐场景**:
- 生产环境诊断
- 需要快速部署
- 无法安装额外依赖
- 离线分析 diagnose_k8s.sh 数据
- 需要详细的排查建议

### 何时使用 v2.0 Go 版本？

✅ **适用场景**:
- 需要在线实时诊断 K8s 集群
- 需要交互式 TUI 界面
- 需要根因分析 (RCA)
- 需要自动修复功能
- 需要成本分析
- 需要镜像安全扫描
- 需要自定义 YAML 规则
- 需要 Kubernetes 原生部署（DaemonSet/Operator）
- 需要跨平台支持（Windows/macOS）
- 需要 SARIF 安全报告集成
- 需要 Grafana 监控集成
- 希望参与开源开发

## 贡献

欢迎贡献！请阅读以下文档了解详情：

- [贡献指南](CONTRIBUTING.md) - 如何参与项目贡献
- [行为准则](CODE_OF_CONDUCT.md) - 社区行为准则
- [安全政策](SECURITY.md) - 安全漏洞报告

### 版本特定文档

- [v2-go 开发指南](v2-go/README.md#开发)
- [v1-bash 使用说明](v1-bash/README.md)

## 许可证

本项目采用 Apache License 2.0 许可证。

---

**版本说明**: v1.0 Bash 版本和 v2.0 Go 版本均为生产可用的稳定版本。v2.0 Go 版本提供了更多高级特性，功能特性已达到 100%。
