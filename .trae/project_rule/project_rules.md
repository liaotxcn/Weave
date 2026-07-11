---
alwaysApply: false
description: 编写golang代码规则
---

# golang aicoding 规则

## 核心原则
- 高效简洁： 优先使用语言内置特性，避免过度冗余逻辑
- 可维护性： 代码结构清晰，适当注释，避免使用魔法数字
- 适当注释： 复杂逻辑和公开函数必须注释
- 根据用户需求，只改动必要模块代码，不影响其他现有功能

## 代码注意
- 尽可能避免使用魔法数字
- 错误处理、错误信息需包含上下文，日志记录清晰(包含关键字段和错误信息)
- 注意并发安全：
如：共享变量保护(使用 `sync.Mutex`、`sync.RWMutex` 或 channel)
goroutine 必须有明确的退出机制（`context.Context` 或 `done channel`）
- 业务逻辑中禁止使用 `panic`，仅用于不可恢复的错误（如 `init()` 失败）
- 数值常量需定义有意义常量名
- 配置值使用常量或配置文件，尽可能避免硬编码
```go
// ❌ 错误：魔法数值
time.Sleep(30 * time.Second)
if status == 3 { }

// ✅ 正确：定义常量
const (
    DefaultTimeout = 30 * time.Second
    StatusActive   = 3
)
time.Sleep(DefaultTimeout)
if status == StatusActive { }
```

## 工作流程
每次编码前执行：
1、充分理解需求
2、定位问题/需求模块范围
3、编写代码
4、添加适当注释
5、检查并进行编译验证
6、输出代码变更前后差异，所影响的逻辑处理等
