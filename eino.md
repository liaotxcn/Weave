# v0.8.*-adk middlewares


本文档介绍 Eino ADK v0.8.* 版本的主要新功能和改进。

## 版本亮点

v0.8 是一个重要的功能增强版本，引入了全新的中间件接口架构，新增多个实用中间件，并提供了增强的可观测性支持。

<table><tbody><tr>
<td>
<strong>🔧 灵活的中间件架构</strong>
全新 ChatModelAgentMiddleware 接口</td><td>
<strong>📊 增强的可观测性</strong>
Agent 级别 Callback 支持</td></tr></tbody></table>

---

## 1. ChatModelAgentMiddleware 接口

> 💡
> **核心更新**: 全新的中间件接口，提供更灵活的 Agent 扩展机制

`ChatModelAgentMiddleware` 是 v0.8 最重要的架构更新，为 `ChatModelAgent` 及基于它构建的 Agent（如 `DeepAgent`）提供统一的扩展点。

**相比 AgentMiddleware 的优势**:

<table>
<tr><td>特性</td><td>AgentMiddleware</td><td>ChatModelAgentMiddleware</td></tr>
<tr><td>扩展性</td><td>封闭</td><td>开放，可实现自定义 handler</td></tr>
<tr><td>Context 传播</td><td>回调只返回 error</td><td>所有方法返回 (ctx, ..., error)</td></tr>
<tr><td>配置管理</td><td>分散在闭包中</td><td>集中在结构体字段中</td></tr>
</table>

**接口方法**:

- `BeforeAgent` - Agent 运行前修改配置
- `BeforeModelRewriteState` - 模型调用前处理状态
- `AfterModelRewriteState` - 模型调用后处理状态
- `WrapInvokableToolCall` - 包装同步工具调用
- `WrapStreamableToolCall` - 包装流式工具调用
- `WrapModel` - 包装模型调用

**使用方式**:

```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Model:    model,
    Handlers: []adk.ChatModelAgentMiddleware{mw1, mw2, mw3},
})
```

详见 [Eino ADK: ChatModelAgentMiddleware](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware)

---

### 1.1 Summarization 中间件

> 💡
> **功能**: 自动对话历史摘要，防止超出模型上下文窗口限制

📚 **详细文档**: [Middleware: Summarization](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_summarization)

当对话历史的 Token 数量超过阈值时，自动调用 LLM 生成摘要，压缩上下文。

**核心能力**:

- 可配置的触发条件（Token 阈值）
- 支持保留最近的用户消息
- 支持记录完整对话历史到文件
- 提供前后处理钩子

**快速开始**:

```go
mw, err := summarization.New(ctx, &summarization.Config{
    Model: chatModel,
    Trigger: &summarization.TriggerCondition{
        ContextTokens: 100000,
    },
})
```

### 1.2 ToolReduction 中间件

> 💡
> **功能**: 工具结果压缩，优化上下文使用效率

📚 **详细文档**: [Middleware: ToolReduction](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_toolreduction)

提供两阶段的工具输出管理：

<table>
<tr><td>阶段</td><td>触发时机</td><td>作用</td></tr>
<tr><td>截断 (Truncation)</td><td>工具返回后</td><td>截断超长输出，保存到文件</td></tr>
<tr><td>清理 (Clear)</td><td>模型调用前</td><td>清理历史工具结果，释放 Token</td></tr>
</table>

**快速开始**:

```go
mw, err := reduction.New(ctx, &reduction.Config{
    Backend:           fsBackend,
    MaxLengthForTrunc: 30000,
    MaxTokensForClear: 50000,
})
```

### 1.3 Filesystem 中间件

> 💡
> **功能**: 文件系统操作工具集

📚 **详细文档**: [Middleware: FileSystem](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_filesystem)

**新增能力**:

- **Grep 功能增强**: 支持完整正则表达式语法
- **新增选项**: `CaseInsensitive`、`EnableMultiline`、`FileType` 过滤
- **自定义工具名**: 所有 filesystem 工具支持自定义名称

### 1.4 Skill 中间件

> 💡
> **功能**: 动态加载和执行 Skill

📚 **详细文档**: [Middleware: Skill](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_skill)

**新增能力**:

- **Context 模式**: 支持 `fork` 和 `isolate` 两种上下文模式
- **自定义配置**: 支持自定义系统提示和工具描述
- **FrontMatter 扩展**: 支持通过 FrontMatter 指定 agent 和 model

### 1.5 PlanTask 中间件

> 💡
> **功能**: 任务规划和执行工具

📚 **详细文档**: [Middleware: PlanTask](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_plantask)

支持 Agent 创建和管理任务计划，适用于需要分步执行的复杂任务场景。

### 1.6 ToolSearch 中间件

> 💡
> **功能**: 工具搜索，支持从大量工具中动态检索

📚 **详细文档**: [Middleware: ToolSearch](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_toolsearch)

当工具数量较多时，通过语义搜索动态选择最相关的工具，避免上下文过载。

### 1.7 PatchToolCalls 中间件

> 💡
> **功能**: 修补悬空的工具调用，确保消息历史完整性

📚 **详细文档**: [Middleware: PatchToolCalls](/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_patchtoolcalls)

扫描消息历史，为缺少响应的工具调用插入占位符消息。适用于工具调用被中断或取消的场景。

**快速开始**:

```go
mw, err := patchtoolcalls.New(ctx, nil)
```

## 2. Agent Callback 支持

> 💡
> **功能**: Agent 级别的回调机制，用于观测和追踪

支持在 Agent 执行的全生命周期中注册回调，实现日志记录、追踪、监控等功能。

**核心类型**:

- `AgentCallbackInput` - 回调输入，包含 Agent 输入或恢复信息
- `AgentCallbackOutput` - 回调输出，包含 Agent 事件流

**使用方式**:

```go
agent.Run(ctx, input, adk.WithCallbacks(
    callbacks.NewHandler(
        callbacks.WithOnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            agentInput := adk.ConvAgentCallbackInput(input)
            // 处理 Agent 启动事件
            return ctx
        }),
        callbacks.WithOnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            agentOutput := adk.ConvAgentCallbackOutput(output)
            // 处理 Agent 完成事件
            return ctx
        }),
    ),
))
```

详见 [Eino ADK: Agent Callback](/zh/docs/eino/core_modules/eino_adk/adk_agent_callback)

---

## 3. Language Setting

> 💡
> **功能**: 全局语言设置

支持全局设置 ADK 的语言偏好，影响内置提示词和消息的语言。

**使用方式**:

```go
adk.SetLanguage(adk.LanguageChinese)  // 设置为中文
adk.SetLanguage(adk.LanguageEnglish)  // 设置为英文（默认）
```

---

## 中间件使用建议

> 💡
> **推荐组合**: 以下中间件可组合使用，覆盖大部分长对话场景

```go
handlers := []adk.ChatModelAgentMiddleware{
    patchMW,          // 1. 修补悬空工具调用
    reductionMW,      // 2. 压缩工具输出
    summarizationMW,  // 3. 摘要对话历史
}

agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Model:    model,
    Handlers: handlers,
})
```

---

## Breaking Changes

> 💡
> 升级到 v0.8 前，请查阅 Breaking Changes 文档了解所有不兼容变更

📚 **完整文档**: [Eino v0.8 不兼容更新](/zh/docs/eino/release_notes_and_migration/eino_v0.8._-adk_middlewares/eino_v0.8_不兼容更新)

**变更概览**:

<table>
<tr><td>类型</td><td>变更项</td></tr>
<tr><td>API 变更</td><td><pre>ShellBackend</pre> → <pre>Shell</pre> 接口重命名</td></tr>
<tr><td>行为变更</td><td><pre>AgentEvent</pre> 发送机制改为 Middleware</td></tr>
<tr><td>行为变更</td><td><pre>ReadRequest.Offset</pre> 从 0-based 改为 1-based</td></tr>
<tr><td>行为变更</td><td><pre>FileInfo.Path</pre> 不再保证为绝对路径</td></tr>
<tr><td>行为变更</td><td><pre>WriteRequest</pre> 文件存在时从报错改为覆盖</td></tr>
<tr><td>行为变更</td><td><pre>GrepRequest.Pattern</pre> 从字面量改为正则表达式</td></tr>
</table>

## 升级指南

详细的迁移步骤和代码示例请参考：[Eino v0.8 不兼容更新](/zh/docs/eino/release_notes_and_migration/eino_v0.8._-adk_middlewares/eino_v0.8_不兼容更新)

**快速检查清单**:

1. 检查是否使用了 `ShellBackend` / `StreamingShellBackend` 接口（需重命名）
2. 检查 `ReadRequest.Offset` 使用（0-based → 1-based）
3. 检查 `GrepRequest.Pattern` 使用（字面量 → 正则表达式，特殊字符需转义）
4. 检查是否依赖 `WriteRequest` 的"文件存在报错"行为
5. 检查是否依赖 `FileInfo.Path` 为绝对路径
6. 如有自定义 ChatModel/Tool Decorator/Wrapper，考虑迁移到 `ChatModelAgentMiddleware`
7. 运行测试验证功能正常


---
Source: https://www.cloudwego.io/zh/docs/eino/release_notes_and_migration/eino_v0.8._-adk_middlewares/



# ChatModelAgentMiddleware


## 概述

## ChatModelAgentMiddleware 接口

`ChatModelAgentMiddleware` 定义了自定义 `ChatModelAgent` 行为的接口。

**重要说明：** 此接口专为 `ChatModelAgent` 及基于它构建的 Agent（如 `DeepAgent`）设计。

> 💡
> ChatModelAgentMiddleware 接口在 v0.8.0 版本引入

### 为什么使用 ChatModelAgentMiddleware 而非 AgentMiddleware？

<table>
<tr><td>特性</td><td>AgentMiddleware (结构体)</td><td>ChatModelAgentMiddleware (接口)</td></tr>
<tr><td>扩展性</td><td>封闭，用户无法添加新方法</td><td>开放，用户可实现自定义 handler</td></tr>
<tr><td>Context 传播</td><td>回调只返回 error</td><td>所有方法返回 <pre>(context.Context, ..., error)</pre></td></tr>
<tr><td>配置管理</td><td>分散在闭包中</td><td>集中在结构体字段中</td></tr>
</table>

### 接口定义

```go
type ChatModelAgentMiddleware interface {
    // BeforeAgent 在每次 agent 运行前调用，允许修改 instruction 和 tools 配置
    BeforeAgent(ctx context.Context, runCtx *ChatModelAgentContext) (context.Context, *ChatModelAgentContext, error)

    // BeforeModelRewriteState 在每次模型调用前调用
    // 返回的 state 会被持久化到 agent 内部状态并传递给模型
    // 返回的 context 会传播到模型调用和后续 handler
    BeforeModelRewriteState(ctx context.Context, state *ChatModelAgentState, mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // AfterModelRewriteState 在每次模型调用后调用
    // 输入的 state 包含模型响应作为最后一条消息
    AfterModelRewriteState(ctx context.Context, state *ChatModelAgentState, mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // WrapInvokableToolCall 用自定义行为包装工具的同步执行
    // 如果不需要包装，返回原始 endpoint 和 nil error
    // 仅对实现了 InvokableTool 的工具调用此方法
    WrapInvokableToolCall(ctx context.Context, endpoint InvokableToolCallEndpoint, tCtx *ToolContext) (InvokableToolCallEndpoint, error)

    // WrapStreamableToolCall 用自定义行为包装工具的流式执行
    // 如果不需要包装，返回原始 endpoint 和 nil error
    // 仅对实现了 StreamableTool 的工具调用此方法
    WrapStreamableToolCall(ctx context.Context, endpoint StreamableToolCallEndpoint, tCtx *ToolContext) (StreamableToolCallEndpoint, error)

    // WrapEnhancedInvokableToolCall 用自定义行为包装增强型工具的同步执行
    WrapEnhancedInvokableToolCall(ctx context.Context, endpoint EnhancedInvokableToolCallEndpoint, tCtx *ToolContext) (EnhancedInvokableToolCallEndpoint, error)

    // WrapEnhancedStreamableToolCall 用自定义行为包装增强型工具的流式执行
    WrapEnhancedStreamableToolCall(ctx context.Context, endpoint EnhancedStreamableToolCallEndpoint, tCtx *ToolContext) (EnhancedStreamableToolCallEndpoint, error)

    // WrapModel 用自定义行为包装聊天模型
    // 如果不需要包装，返回原始 model 和 nil error
    // 在请求时调用，每次模型调用前都会执行
    WrapModel(ctx context.Context, m model.BaseChatModel, mc *ModelContext) (model.BaseChatModel, error)
}
```

### 使用 BaseChatModelAgentMiddleware

嵌入 `*BaseChatModelAgentMiddleware` 以获得默认的空操作实现：

```go
type MyHandler struct {
    *adk.BaseChatModelAgentMiddleware
}

func (h *MyHandler) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
    return ctx, state, nil
}
```

---

## 工具调用端点类型

工具包装使用函数类型而非接口，更清晰地表达了包装的意图：

```go
// InvokableToolCallEndpoint 是同步工具调用的函数签名
type InvokableToolCallEndpoint func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)

// StreamableToolCallEndpoint 是流式工具调用的函数签名
type StreamableToolCallEndpoint func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error)

// EnhancedInvokableToolCallEndpoint 是增强型同步工具调用的函数签名
type EnhancedInvokableToolCallEndpoint func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.ToolResult, error)

// EnhancedStreamableToolCallEndpoint 是增强型流式工具调用的函数签名
type EnhancedStreamableToolCallEndpoint func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error)
```

### 为什么使用分离的端点类型？

之前的 `ToolCall` 接口同时包含 `InvokableRun` 和 `StreamableRun`，但大多数工具只实现其中一个。
分离的端点类型使得：

- 只有当工具实现相应接口时才调用对应的包装方法
- wrapper 作者更清晰的契约
- 关于实现哪个方法没有歧义

---

## ChatModelAgentContext

`ChatModelAgentContext` 包含在每次 `ChatModelAgent` 运行前传递给 handler 的运行时信息。

```go
type ChatModelAgentContext struct {
    // Instruction 是当前 Agent 执行的指令
    // 包括 agent 配置的指令、框架和 AgentMiddleware 追加的额外指令，
    // 以及之前 BeforeAgent handler 应用的修改
    Instruction string

    // Tools 是当前为 Agent 执行配置的原始工具（无任何 wrapper 或 tool middleware）
    // 包括 AgentConfig 中传入的工具、框架隐式添加的工具（如 transfer/exit 工具），
    // 以及 middleware 已添加的其他工具
    Tools []tool.BaseTool

    // ReturnDirectly 是当前配置为使 Agent 直接返回的工具名称集合
    ReturnDirectly map[string]bool
}
```

---

## ChatModelAgentState

`ChatModelAgentState` 表示对话过程中聊天模型 agent 的状态。这是 `ChatModelAgentMiddleware` 和 `AgentMiddleware` 回调的主要状态类型。

```go
type ChatModelAgentState struct {
    // Messages 包含当前对话会话中的所有消息
    Messages []Message
}
```

---

## ToolContext

`ToolContext` 提供被包装工具的元数据。在请求时创建，包含当前工具调用的信息。

```go
type ToolContext struct {
    // Name 是工具名称
    Name string

    // CallID 是此特定工具调用的唯一标识符
    CallID string
}
```

### 使用示例：工具调用包装

```go
func (h *MyHandler) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
        log.Printf("Tool %s (call %s) starting with args: %s", tCtx.Name, tCtx.CallID, argumentsInJSON)
        
        result, err := endpoint(ctx, argumentsInJSON, opts...)
        
        if err != nil {
            log.Printf("Tool %s failed: %v", tCtx.Name, err)
            return "", err
        }
        
        log.Printf("Tool %s completed with result: %s", tCtx.Name, result)
        return result, nil
    }, nil
}
```

---

## ModelContext

`ModelContext` 包含传递给 `WrapModel` 的上下文信息。在请求时创建，包含当前模型调用的工具配置。

```go
type ModelContext struct {
    // Tools 是当前配置给 agent 的工具列表
    // 在请求时填充，包含将发送给模型的工具
    Tools []*schema.ToolInfo

    // ModelRetryConfig 包含模型的重试配置
    // 在请求时从 agent 的 ModelRetryConfig 填充
    // 用于 EventSenderModelWrapper 适当地包装流错误
    ModelRetryConfig *ModelRetryConfig
}
```

### 使用示例：模型包装

```go
func (h *MyHandler) WrapModel(ctx context.Context, m model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
    return &myModelWrapper{
        inner: m,
        tools: mc.Tools,
    }, nil
}

type myModelWrapper struct {
    inner model.BaseChatModel
    tools []*schema.ToolInfo
}

func (w *myModelWrapper) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    log.Printf("Model called with %d tools", len(w.tools))
    return w.inner.Generate(ctx, msgs, opts...)
}

func (w *myModelWrapper) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    return w.inner.Stream(ctx, msgs, opts...)
}
```

---

## 运行时本地存储 API

`SetRunLocalValue`、`GetRunLocalValue` 和 `DeleteRunLocalValue` 提供在当前 agent Run() 调用期间存储、获取和删除值的能力。

```go
// SetRunLocalValue 设置一个在当前 agent Run() 调用期间持久化的键值对
// 值的作用域限于此特定执行，不会在不同的 Run() 调用或 agent 实例之间共享
//
// 存储在这里的值与中断/恢复周期兼容 - 它们会被序列化并在 agent 恢复时还原
// 对于自定义类型，必须在 init() 函数中使用 schema.RegisterName[T]() 注册以确保正确序列化
//
// 此函数只能在 agent 执行期间从 ChatModelAgentMiddleware 内部调用
// 如果在 agent 执行上下文之外调用，返回错误
func SetRunLocalValue(ctx context.Context, key string, value any) error

// GetRunLocalValue 获取在当前 agent Run() 调用期间设置的值
// 值的作用域限于此特定执行，不会在不同的 Run() 调用或 agent 实例之间共享
//
// 通过 SetRunLocalValue 存储的值与中断/恢复周期兼容 - 它们会被序列化并在 agent 恢复时还原
// 对于自定义类型，必须在 init() 函数中使用 schema.RegisterName[T]() 注册以确保正确序列化
//
// 此函数只能在 agent 执行期间从 ChatModelAgentMiddleware 内部调用
// 如果找到值返回 (value, true, nil)，如果未找到返回 (nil, false, nil)，
// 如果在 agent 执行上下文之外调用返回错误
func GetRunLocalValue(ctx context.Context, key string) (any, bool, error)

// DeleteRunLocalValue 删除在当前 agent Run() 调用期间设置的值
//
// 此函数只能在 agent 执行期间从 ChatModelAgentMiddleware 内部调用
// 如果在 agent 执行上下文之外调用，返回错误
func DeleteRunLocalValue(ctx context.Context, key string) error
```

### 使用示例：跨 handler 点共享数据

```go
func init() {
    schema.RegisterName[*MyCustomData]("my_package.MyCustomData")
}

type MyCustomData struct {
    Count int
    Name  string
}

type MyHandler struct {
    *adk.BaseChatModelAgentMiddleware
}

func (h *MyHandler) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, argumentsInJSON, opts...)
        
        data := &MyCustomData{Count: 1, Name: tCtx.Name}
        if err := adk.SetRunLocalValue(ctx, "my_handler.last_tool", data); err != nil {
            log.Printf("Failed to set run local value: %v", err)
        }
        
        return result, err
    }, nil
}

func (h *MyHandler) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
    if val, found, err := adk.GetRunLocalValue(ctx, "my_handler.last_tool"); err == nil && found {
        if data, ok := val.(*MyCustomData); ok {
            log.Printf("Last tool was: %s (count: %d)", data.Name, data.Count)
        }
    }
    return ctx, state, nil
}
```

---

## SendEvent API

`SendEvent` 允许在 agent 执行期间向事件流发送自定义 `AgentEvent`。

```go
// SendEvent 在 agent 执行期间向事件流发送自定义 AgentEvent
// 允许 ChatModelAgentMiddleware 实现发出自定义事件，
// 这些事件将被遍历 agent 事件流的调用者接收
//
// 此函数只能在 agent 执行期间从 ChatModelAgentMiddleware 内部调用
// 如果在 agent 执行上下文之外调用，返回错误
func SendEvent(ctx context.Context, event *AgentEvent) error
```

---

## State 类型（即将弃用）

`State` 保存 agent 运行时状态，包括消息和用户可扩展存储。

**⚠️ 弃用警告：** 此类型将在 v1.0.0 中设为未导出。请在 `ChatModelAgentMiddleware` 和 `AgentMiddleware` 回调中使用 `ChatModelAgentState`。不建议直接使用 `compose.ProcessState[*State]`，该用法将在 v1.0.0 中停止工作；请使用 handler API。

```go
type State struct {
    Messages []Message
    extra    map[string]any  // 未导出，通过 SetRunLocalValue/GetRunLocalValue 访问

    // 以下为内部字段 - 请勿直接访问
    // 为与现有 checkpoint 向后兼容而保持导出
    ReturnDirectlyToolCallID string
    ToolGenActions           map[string]*AgentAction
    AgentName                string
    RemainingIterations      int

    internals map[string]any
}
```

---

## 架构图

下图展示了 `ChatModelAgentMiddleware` 在 `ChatModelAgent` 执行过程中的工作原理：

```
Agent.Run(input)
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  BeforeAgent(ctx, *ChatModelAgentContext)                               │
│    输入: 当前 Instruction、Tools 等 Agent 运行环境                          │
│    输出: 修改后的 Agent 运行环境                                         │
│    作用: Run 开始时调用一次，修改整个 Run 生命周期的配置                   │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          ReAct Loop                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                                                                   │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │  BeforeModelRewriteState(ctx, *ChatModelAgentState, *MC)    │  │  │
│  │  │    输入: 消息历史等持久化状态，以及 Model 运行环境            │  │  │
│  │  │    输出: 修改后的持久化状态，返回新 ctx                       │  │  │
│  │  │    作用: 修改跨 iteration 的持久化状态（主要是消息列表）      │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                            │                                      │  │
│  │                            ▼                                      │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │  WrapModel(ctx, BaseChatModel, *ModelContext)               │  │  │
│  │  │    输入: 被 wrap 的 ChatModel，以及 Model 运行环境           │  │  │
│  │  │    输出: 包装后的 Model (洋葱模型)                           │  │  │
│  │  │    作用: 修改单次 Model 请求的输入、输出和配置               │  │  │
│  │  │                         │                                   │  │  │
│  │  │                         ▼                                   │  │  │
│  │  │                 ┌───────────────┐                           │  │  │
│  │  │                 │    Model      │                           │  │  │
│  │  │                 │ Generate/Stream│                          │  │  │
│  │  │                 └───────────────┘                           │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                            │                                      │  │
│  │                            ▼                                      │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │  AfterModelRewriteState(ctx, *ChatModelAgentState, *MC)     │  │  │
│  │  │    输入: 消息历史等持久化状态（含 Model 响应），             │  │  │
│  │  │          以及 Model 运行环境                                │  │  │
│  │  │    输出: 修改后的持久化状态                                  │  │  │
│  │  │    作用: 修改跨 iteration 的持久化状态（主要是消息列表）     │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                            │                                      │  │
│  │                            ▼                                      │  │
│  │                  ┌──────────────────┐                             │  │
│  │                  │ Model 返回内容?   │                            │  │
│  │                  └──────────────────┘                             │  │
│  │                     │            │                                │  │
│  │            最终响应 │            │ ToolCalls                      │  │
│  │                     │            ▼                                │  │
│  │                     │  ┌─────────────────────────────────────┐    │  │
│  │                     │  │  WrapInvokableToolCall / WrapStream │    │  │
│  │                     │  │  ableToolCall(ctx, endpoint, *TC)   │    │  │
│  │                     │  │    输入: 被 wrap 的 Tool 以及       │    │  │
│  │                     │  │          Tool 运行环境              │    │  │
│  │                     │  │    输出: 包装后的 endpoint (洋葱模型)│    │  │
│  │                     │  │    作用: 修改单次 Tool 请求的       │    │  │
│  │                     │  │          输入、输出和配置           │    │  │
│  │                     │  │                  │                  │    │  │
│  │                     │  │                  ▼                  │    │  │
│  │                     │  │          ┌─────────────┐            │    │  │
│  │                     │  │          │ Tool.Run()  │            │    │  │
│  │                     │  │          └─────────────┘            │    │  │
│  │                     │  └─────────────────────────────────────┘    │  │
│  │                     │            │                                │  │
│  │                     │            │ (结果加入 Messages)            │  │
│  │                     │            │                                │  │
│  │                     │  ┌─────────┘                                │  │
│  │                     │  │                                          │  │
│  │                     │  └──────────► 继续循环                      │  │
│  │                     │                                             │  │
│  └─────────────────────┼─────────────────────────────────────────────┘  │
│                        │                                                │
│                        ▼                                                │
│               循环直到完成或达到 maxIterations                           │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
                          Agent.Run() 结束
```

### Handler 方法说明

<table>
<tr><td>方法</td><td>输入</td><td>输出</td><td>作用范围</td></tr>
<tr><td><pre>BeforeAgent</pre></td><td>Agent 运行环境 (<pre>*ChatModelAgentContext</pre>)</td><td>修改后的 Agent 运行环境</td><td>整个 Run 生命周期，仅调用一次</td></tr>
<tr><td><pre>BeforeModelRewriteState</pre></td><td>持久化状态 + Model 运行环境</td><td>修改后的持久化状态</td><td>跨 iteration 的持久化状态（消息列表）</td></tr>
<tr><td><pre>WrapModel</pre></td><td>被 wrap 的 ChatModel + Model 运行环境</td><td>包装后的 Model</td><td>单次 Model 请求的输入、输出和配置</td></tr>
<tr><td><pre>AfterModelRewriteState</pre></td><td>持久化状态（含响应）+ Model 运行环境</td><td>修改后的持久化状态</td><td>跨 iteration 的持久化状态（消息列表）</td></tr>
<tr><td><pre>WrapInvokableToolCall</pre></td><td>被 wrap 的 Tool + Tool 运行环境</td><td>包装后的 endpoint</td><td>单次 Tool 请求的输入、输出和配置</td></tr>
<tr><td><pre>WrapStreamableToolCall</pre></td><td>被 wrap 的 Tool + Tool 运行环境</td><td>包装后的 endpoint</td><td>单次 Tool 请求的输入、输出和配置</td></tr>
</table>

---

## 执行顺序

### Model 调用生命周期（从外到内的 wrapper 链）

1. `AgentMiddleware.BeforeChatModel`（hook，在模型调用前运行）
2. `ChatModelAgentMiddleware.BeforeModelRewriteState`（hook，可在模型调用前修改状态）
3. `retryModelWrapper`（内部 - 失败时重试，如已配置）
4. `eventSenderModelWrapper` 预处理（内部 - 准备事件发送）
5. `ChatModelAgentMiddleware.WrapModel` 预处理（wrapper，在请求时包装，先注册的先运行）
6. `callbackInjectionModelWrapper`（内部 - 如未启用则注入回调）
7. `Model.Generate/Stream`
8. `callbackInjectionModelWrapper` 后处理
9. `ChatModelAgentMiddleware.WrapModel` 后处理（wrapper，先注册的后运行）
10. `eventSenderModelWrapper` 后处理（内部 - 发送模型响应事件）
11. `retryModelWrapper` 后处理（内部 - 处理重试逻辑）
12. `ChatModelAgentMiddleware.AfterModelRewriteState`（hook，可在模型调用后修改状态）
13. `AgentMiddleware.AfterChatModel`（hook，在模型调用后运行）

### Tool 调用生命周期（从外到内）

1. `eventSenderToolHandler`（内部 ToolMiddleware - 在所有处理后发送工具结果事件）
2. `ToolsConfig.ToolCallMiddlewares`（ToolMiddleware）
3. `AgentMiddleware.WrapToolCall`（ToolMiddleware）
4. `ChatModelAgentMiddleware.WrapInvokableToolCall/WrapStreamableToolCall`（在请求时包装，先注册的在最外层）
5. `Tool.InvokableRun/StreamableRun`

---

## 迁移指南

### 从 AgentMiddleware 迁移到 ChatModelAgentMiddleware

**之前（AgentMiddleware）：**

```go
middleware := adk.AgentMiddleware{
    BeforeChatModel: func(ctx context.Context, state *adk.ChatModelAgentState) error {
        return nil
    },
}
```

**之后（ChatModelAgentMiddleware）：**

```go
type MyHandler struct {
    *adk.BaseChatModelAgentMiddleware
}

func (h *MyHandler) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
    newCtx := context.WithValue(ctx, myKey, myValue)
    return newCtx, state, nil
}
```

### 从 compose.ProcessState[*State] 迁移

**之前：**

```go
compose.ProcessState(ctx, func(_ context.Context, st *adk.State) error {
    st.Extra["myKey"] = myValue
    return nil
})
```

**之后（使用 SetRunLocalValue/GetRunLocalValue）：**

```go
if err := adk.SetRunLocalValue(ctx, "myKey", myValue); err != nil {
    return ctx, state, err
}

if val, found, err := adk.GetRunLocalValue(ctx, "myKey"); err == nil && found {
}
```


---
Source: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/