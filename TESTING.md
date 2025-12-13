# kudig.sh 测试说明

## 测试环境要求

由于 `kudig.sh` 是一个Bash脚本，需要在Linux环境或具有Bash环境的系统上运行。

### 在Linux/Unix系统上测试

1. **准备诊断数据**：
```bash
# 在Kubernetes节点上收集诊断数据
sudo ./diagnose_k8s.sh

# 会生成类似 /tmp/diagnose_1702468800 的目录
```

2. **运行kudig.sh**：
```bash
# 添加执行权限
chmod +x kudig.sh

# 基本测试
./kudig.sh /tmp/diagnose_1702468800

# 详细模式测试
./kudig.sh --verbose /tmp/diagnose_1702468800

# JSON格式测试
./kudig.sh --json /tmp/diagnose_1702468800

# 保存到文件测试
./kudig.sh -o report.txt /tmp/diagnose_1702468800
```

3. **验证退出码**：
```bash
./kudig.sh /tmp/diagnose_1702468800
echo "Exit code: $?"
# 0 = 无异常
# 1 = 有警告/提示
# 2 = 有严重异常
```

### 在Windows WSL上测试

如果使用Windows系统，可以通过WSL (Windows Subsystem for Linux) 运行：

```bash
# 在WSL中
cd /mnt/c/Users/Allen/Documents/GitHub/kudig.sh
chmod +x kudig.sh

# 创建测试目录（模拟诊断数据）
./create_test_data.sh

# 运行测试
./kudig.sh ./test_diagnose_dir
```

### 使用Git Bash测试

如果安装了Git for Windows，可以使用Git Bash：

```bash
cd /c/Users/Allen/Documents/GitHub/kudig.sh
bash kudig.sh --help
```

## 功能验证清单

- [ ] 帮助信息显示正常 (`--help`)
- [ ] 版本信息显示正常 (`--version`)
- [ ] 能够正确解析诊断目录
- [ ] 系统资源检测功能正常
- [ ] 进程服务检测功能正常
- [ ] 网络检测功能正常
- [ ] 内核检测功能正常
- [ ] 容器运行时检测功能正常
- [ ] Kubernetes组件检测功能正常
- [ ] 时间同步检测功能正常
- [ ] 配置检测功能正常
- [ ] 异常去重功能正常
- [ ] 异常排序功能正常
- [ ] 文本格式输出正常
- [ ] JSON格式输出正常
- [ ] 文件保存功能正常
- [ ] 退出码正确

## 预期输出示例

### 无异常情况

```
=== Kubernetes节点诊断异常报告 ===
诊断时间: 2024-12-13 10:30:00
节点信息: k8s-node-01
分析目录: /tmp/diagnose_1702468800

✓ 未检测到异常

节点状态良好！
```

退出码: 0

### 有异常情况

```
=== Kubernetes节点诊断异常报告 ===
诊断时间: 2024-12-13 10:30:00
节点信息: k8s-node-01
分析目录: /tmp/diagnose_1702468800

-------------------------------------------
【严重级别】异常项
-------------------------------------------
[严重] Kubelet服务未运行 | KUBELET_SERVICE_DOWN
  详情: kubelet.service状态为failed
  位置: daemon_status/kubelet_status

-------------------------------------------
异常统计
-------------------------------------------
严重: 1项
警告: 0项
提示: 0项
总计: 1项
```

退出码: 2

## 自动化测试脚本

可以创建一个自动化测试脚本来验证各个功能：

```bash
#!/bin/bash

echo "开始kudig.sh功能测试..."

# 测试1: 帮助信息
echo "测试1: 帮助信息"
./kudig.sh --help
if [ $? -eq 0 ]; then
    echo "✓ 帮助信息测试通过"
else
    echo "✗ 帮助信息测试失败"
fi

# 测试2: 版本信息
echo "测试2: 版本信息"
./kudig.sh --version
if [ $? -eq 0 ]; then
    echo "✓ 版本信息测试通过"
else
    echo "✗ 版本信息测试失败"
fi

# 测试3: 空目录测试
echo "测试3: 空目录测试"
mkdir -p /tmp/test_empty_diagnose
./kudig.sh /tmp/test_empty_diagnose
rm -rf /tmp/test_empty_diagnose

# 测试4: JSON输出格式
echo "测试4: JSON输出格式"
./kudig.sh --json /tmp/diagnose_* | python -m json.tool > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ JSON格式测试通过"
else
    echo "✗ JSON格式测试失败"
fi

echo "测试完成！"
```

## 注意事项

1. 脚本需要在Linux环境或支持Bash的环境下运行
2. 诊断目录必须是由 `diagnose_k8s.sh` 生成的完整目录
3. 脚本不需要root权限，普通用户即可运行
4. 脚本只读取诊断数据，不会修改任何文件

## 常见问题

**Q: 在Windows上如何测试？**
A: 推荐使用WSL (Windows Subsystem for Linux) 或Git Bash。

**Q: 如果诊断目录不完整会怎样？**
A: 脚本会显示警告但继续分析可用的文件，不会中断执行。

**Q: 为什么某些检测项没有结果？**
A: 可能是对应的日志文件在诊断数据中缺失，这是正常的。

**Q: 如何验证JSON输出格式正确？**
A: 可以使用 `jq` 或 `python -m json.tool` 验证JSON格式。
