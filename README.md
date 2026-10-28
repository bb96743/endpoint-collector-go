# endpoint-collector-go

端点信息采集工具集（Go）——进程采集与软件资产采集（含文件 CRC 指纹）的跨平台命令行实现。

业务背景来自多年终端管理系统（CEMS）研发经验：采集策略是端点管理的核心策略类型，本仓库将其中两个采集工具用 Go 重构为可独立运行的跨平台原型。

## 工具

| 工具 | 功能 |
|---|---|
| `procscan` | 枚举系统进程（PID / PPID / 进程名）。Windows 走 `CreateToolhelp32Snapshot` 系统调用；Linux 解析 `/proc` 伪文件系统 |
| `softscan` | ① `-list`：已安装软件清单（Windows 读注册表 Uninstall 键含 WOW6432Node；Linux 解析 dpkg 包库）② `-crc <dir>`：目录内全部文件流式计算 CRC-32，goroutine 池并发，1MB 复用缓冲常量内存，被占用/无权限文件跳过不中断 |

## 构建

```bash
# Windows amd64（本机）
go build -ldflags="-s -w" -o bin/procscan.exe ./cmd/procscan
go build -ldflags="-s -w" -o bin/softscan.exe ./cmd/softscan

# Linux arm64（信创鲲鹏 / 飞腾，交叉编译直出）
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/procscan_linux_arm64 ./cmd/procscan
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/softscan_linux_arm64 ./cmd/softscan
```

## 实测（Windows 11 / amd64）

| 项 | 结果 |
|---|---|
| 进程枚举 | 224 个进程，全量输出 <50ms |
| 软件清单 | 注册表 Uninstall 键读取 63 款软件 |
| CRC 全量 | 13,261 个文件 / 244 MB，24.4s，0 失败（1MB 流式缓冲，内存恒定） |
| 单二进制体积 | 1.7 ~ 2.2 MB，无任何运行时依赖 |

## 设计要点

- **常量内存**：`io.CopyBuffer` + 固定 1MB 缓冲，GB 级安装包不整读内存
- **并发受控**：goroutine 池 = CPU 核数 × 2，channel 分发任务，避免打爆磁盘 IO
- **失败容忍**：锁定 / 无权限文件计入 skipped，采集整体不中断——"采全"优先于"采完美"
- **CRC 而非 MD5**：需求是变更检测不是防篡改，CRC-32 快一个数量级且标准库内置；`hash.Hash` 接口一致，升级 SHA-256 零改动
- **信创友好**：`GOOS=linux GOARCH=arm64` 一键交叉编译，鲲鹏 / 飞腾直接运行

## License

MIT
