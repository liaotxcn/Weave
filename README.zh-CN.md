# A highly efficient, secure, and stable application development platform with excellent performance, easy scalability, and deep integration of AI capabilities such as LLM, AI Chat, RAG, and Agents.

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Microkernel-Layered-6BA539?style=for-the-badge" alt="Architecture">
  <img src="https://img.shields.io/badge/LLM--Agent--MCP-74AA9C?style=for-the-badge&logo=brain&logoColor=white" alt="LLM-Agent-MCP">
  <img src="https://img.shields.io/badge/AIChat--RAG-FF6F00?style=for-the-badge&logo=ai&logoColor=white" alt="AIChat-RAG">
  <img src="https://img.shields.io/badge/Cloud_Native-3371E3?style=for-the-badge&logo=Docker&logoColor=white" alt="Cloud Native">
</div>

## 📋 项目简介

Weave (可译为"编织") 从一根简单的线到一幅复杂的锦缎，编织就是从简单到复杂的创造过程。 基于 Golang 开发的高性能、高效率、高扩展易扩展的**赋能工具/AI应用研发平台**，专为高效构建稳定可靠智能应用而设计。采用微内核+分层架构设计，允许开发者高效开发并且轻松集成管理各种工具/服务，同时保持系统的高性能和可扩展性。

<img width="660" height="265" alt="image" src="https://github.com/user-attachments/assets/1c4b8e34-aa4b-496a-8a27-ec7f212e0cc7" />

Weave 核心优势在于其稳定可靠的 **AI功能研发栈+插件工具体系**，深度集成 LLM（大语言模型）、AIChat（智能聊天）、Agent（智能代理）、RAG（检索增强生成）等核心 AI 能力，提供统一的开发框架和标准化接口，让开发者无需关注底层复杂实现，即可快速构建强大的智能应用。

主要应用场景包括：
- 工具/应用研发聚合
- 数据/服务流转中台
- API网关与服务编排
- AI应用研发-构建 LLM、AIChat、Agent、RAG 智能应用
- 自动化工作流-结合 ETL、DevOps、Agent 复杂任务自动化执行

---

## 🏗️ 整体架构

<img width="1562" height="545" alt="image" src="https://github.com/user-attachments/assets/e56f5d3b-b9e4-417a-992e-2997991aa2ad" />

Weave 采用微内核+分层架构设计模式，充分结合两种架构的优势，保障系统可用性，实现了高度的灵活性、可扩展性和良好的性能。

### 微内核与分层架构的融合

Weave 在微内核架构的基础上，融入了分层架构的设计思想，形成了一套完整且高效灵活的架构体系：

- 微内核架构（插件体系）：提供插件管理、生命周期控制和插件间通信机制
- 分层架构（核心系统）：将核心功能按关注点分离，形成清晰的层次结构

### 微内核架构组成

- 核心内核（Core Kernel）：提供基础运行时环境、插件管理、配置管理、日志服务、安全机制等基础功能
- 插件系统（Plugin System）：插件管理器负责插件的注册、生命周期管理、依赖解析和冲突检测
- 扩展插件（Extensions）：通过插件接口集成到核心系统，实现各种业务功能

### 分层架构组成

- 接口层：处理HTTP请求，包括路由管理和控制器
- 业务层：包含核心业务逻辑和插件系统
- 数据层：负责数据存储和访问
- 基础设施层：提供日志、配置、安全等服务

### 架构特点

- 松耦合设计：核心系统与插件之间通过定义良好的接口通信，降低模块间依赖
- 热插拔能力：插件可在运行时动态加载和卸载，无需重启系统
- 功能隔离：每个插件独立封装功能，拥有自己的命名空间和路由前缀
- 依赖与冲突管理：内置依赖解析和冲突检测机制，确保插件间和谐共存
- 统一接口：所有插件实现相同的`Plugin`接口，标准化开发流程
- 可扩展性：系统功能可按需扩展，无需修改内核代码
- 层次清晰：核心系统采用分层设计，代码组织合理，易于维护和扩展
- 高性能：分层设计优化了请求处理流程，提高系统响应速度

系统的核心是高效稳定的插件机制与服务聚合，允许功能模块以插件/服务形式独立开发和部署，同时通过统一的接口进行交互。整体架构设计注重模块化、可扩展性和高性能。

---

## 🌟 项目特点

### 🏗️ 微内核+分层架构
- **核心稳定与层次清晰**：核心系统保持最小化，分层设计使代码组织合理，易于维护和扩展
- **功能扩展灵活**：通过插件机制按需扩展系统功能，无需修改内核代码
- **低耦合高内聚**：系统组件间松耦合，便于维护和升级
- **热插拔能力**：插件可在运行时动态加载和卸载，无需重启系统
- **功能隔离与统一管理**：每个插件独立封装功能，拥有自己的命名空间和路由前缀，同时核心服务通过分层架构统一管理
- **依赖与冲突管理**：内置依赖解析和冲突检测机制，确保插件间和谐共存
- **统一接口**：所有插件实现相同的`Plugin`接口，标准化开发流程
- **高性能**：分层设计优化了请求处理流程，提高系统响应速度

### 🚀 高性能/效率
- 基于 Gin 框架构建，性能高效，并发能力强
- 数据库连接池优化，支持高并发访问
- 模块化架构设计，代码结构清晰，易于维护和扩展
- 支持环境变量覆盖，便于不同环境配置
- 高效路由管理，支持动态路由和参数绑定
- 分层架构优化了请求处理流程，提高系统响应速度

### 🔌 插件化易扩展
- 统一的插件接口设计，支持热插拔
- 插件管理器统一注册、管理和执行插件
- 插件可独立注册路由，拥有独立命名空间
- 插件依赖和冲突检测机制
- 脚手架工具便捷生成插件框架代码
- 示例插件展示完整插件开发流程

### 🧠 AI 深度集成
Weave 提供了完整的 AI 研发功能栈，深度集成 LLM、AIChat、Agent 和 RAG 等核心 AI 技术，为开发者提供一站式的智能应用开发体验。

- **完整的 AI 功能栈**：集成 LLM（大语言模型）、AIChat（智能聊天）、Agent（智能代理）、RAG（检索增强生成）等核心 AI 能力，覆盖从基础模型调用到复杂智能应用开发的全流程
- **多模型支持**：兼容 OpenAI、Ollama、ModelScope 等主流 LLM 平台，支持动态切换模型和平台，满足不同场景需求
- **完善AIChat服务**：多平台多模型多模态、流式响应支持，上下文优化、摘要提取、对话管理等功能
- **智能代理框架**：提供完整的 Agent 开发框架，支持工具调用、任务规划、复杂工作流自动化和多轮对话管理
- **高效 RAG 系统**：基于 RedisSearch 实现高性能向量检索，支持多格式文档处理（文本、PDF、Markdown 等），提升生成内容的准确性和相关性
- **灵活的嵌入模型**：支持自定义嵌入模型和检索参数，适应不同业务场景的向量表示需求
- **架构无缝集成**：AI 能力可作为服务聚合或插件使用，便于快速构建和扩展智能应用

### 🔒 安全可靠
- 基于 JWT 的认证授权系统
- 完善的 CSRF 保护机制
- 基于令牌桶算法的限流中间件
- 密码哈希存储与验证
- 详细的登录历史记录
- 统一的错误处理中间件
- 支持 HTTPS (可在配置中开启)
- 分层架构将安全机制统一封装在基础设施层，便于统一管理和维护

### 📊 可观测性
- 集成结构化日志系统 (zap)
- 健康检查接口，监控系统状态
- 详细的请求/响应日志
- 支持自定义监控指标
- 分层架构将监控功能独立封装，确保系统各层运行状态的可观测性
- 集成 Prometheus 和 Grafana 监控系统，提供可视化仪表盘
- 支持自定义告警规则配置

### 🚀 开发友好
- 完整的插件开发文档和示例
- 插件脚手架工具，快速生成插件模板
- 支持本地开发和 Docker 部署
- 项目结构和代码规范
- 模块清晰、接口标准化，便于开发维护

---

## 📂 项目结构

Weave 采用微内核+分层架构，项目结构清晰地反映了这一设计理念。核心系统采用分层组织，功能扩展则通过插件机制/服务聚合实现

```
├── Dockerfile           # Docker构建文件
├── Makefile             # 构建脚本      
├── config/              # 配置管理
├── controllers/         # API控制器[接口层]
├── docker-compose.yaml  # Docker Compose配置
├── docs/                # 项目文档       
├── main.go              
├── middleware/          # 中间件
├── models/              # 数据模型[数据层]
├── pkg/                 # 公共包[基础设施层]
├── plugins/             # 插件系统[微内核架构核心]
│   ├── core/                 # 核心插件功能
│   ├── examples/             # 示例插件
│   ├── features/             # 功能插件(可扩展)
│   ├── init.go               # 插件初始化
│   ├── loader/               # 插件加载器
│   ├── templates/            # 插件模板
│   └── watcher/              # 插件监控
├── routers/             # 路由定义注册
├── services/            # 服务聚合
│   ├── aichat/          # aichat 服务
│       ├── chat_web/
│       ├── cmd/         # 启动入口
│       ├── internal/    # 核心实现
│       ├── pkg/         
│   ├── rag/                  # RAG 服务
│   ├── email/                # 邮件服务
│   └── extended...           # 可扩展服务
├── test/                # 单元/集成测试
├── tools/               # 开发工具
├── utils/               # 工具函数
└── web/                 # 前端代码
```

---

## 🧩 核心组件

### 🔌 插件系统 - 微内核架构的核心实现
插件系统是Weave的重要组件，负责插件的注册、加载、卸载和生命周期管理。它实现了一套完整的插件机制，使系统能够以插件形式扩展功能。在微内核+分层架构中，插件系统连接了核心内核和各种业务扩展。

- **完整的生命周期管理**：从插件的初始化、注册、激活到关闭的全生命周期管理
- **自动依赖解析**：通过 `GetDependencies()` 方法自动解析插件间依赖关系
- **冲突检测机制**：通过 `GetConflicts()` 方法避免插件间功能冲突
- **路由自动注册**：支持两种路由注册方式，特别是推荐的 `GetRoutes()` 方法更符合微内核架构的设计理念
- **命名空间隔离**：每个插件拥有独立的命名空间，避免资源冲突
- **统一的中间件管理**：支持全局和插件级别的中间件配置

#### 两种路由注册方式的对比
| 特性 | GetRoutes 方法（推荐） | RegisterRoutes 方法（兼容性保留） |
|------|-----------------------|-----------------------------------|
| 路由定义 | 使用 Route 结构体数组 | 直接操作 gin.Engine 对象 |
| 元数据支持 | ✅ 完整支持 | ❌ 不支持 |
| 自动路由组 | ✅ 自动创建 | ❌ 需要手动创建 |
| 中间件管理 | ✅ 支持全局和路由级别 | ❌ 需要手动添加 |
| 文档生成 | ✅ 支持自动生成 API 文档 | ❌ 不支持 |

插件管理器负责插件的整个生命周期管理，包括注册、注销、查询和执行插件功能。

### 🧩 服务聚合
服务聚合是 Weave 在微内核+分层架构基础上的重要扩展能力，提供了将多种服务、数据源和功能进行统一管理和调用的机制。强化AI服务聚合能力，提供了完整的 LLM、AIChat、Agent、RAG 服务体系。

#### 🧠 LLM/AIChat 服务
Weave 的 LLM 服务提供了统一的大语言模型接入和管理能力，让开发者可以轻松调用和切换不同平台的模型，无需关注底层实现细节。
- **多平台兼容，动态模型切换**：支持 OpenAI、Ollama、ModelScope 等主流 LLM 平台，快速切换不同的 LLM 模型和平台
- **标准化接口**：提供统一的 API 接口，易于开发维护
- **流式响应支持**：支持实时流式响应，提升用户交互体验
- **上下文优化**：优化上下文保持和管理，提升对话连贯性
- **摘要提取**：自动提取对话摘要，快速获取关键信息
- **多模态支持**：部分模型支持文本、图像等多模态输入输出
- **模型缓存机制**：内置缓存机制，减少重复请求，提升性能并降低成本

#### 🤖 Agent 服务
Weave 的 Agent 服务提供了完整的智能代理开发和运行框架，支持工具调用、任务规划、内存管理和复杂工作流自动化，让开发者可以快速构建具备自主决策能力的智能应用。
- **工具调用能力**：支持 Agent 调用各种内部工具和外部服务，扩展智能能力边界
- **智能任务规划**：具备自动任务分解、子任务规划和执行路径优化能力
- **灵活内存管理**：支持短期记忆（会话上下文）和长期记忆（知识库）管理，提升决策连续性
- **复杂多轮对话**：支持复杂的多轮对话交互，理解和维护对话上下文
- **工作流自动化**：可实现复杂业务流程的自动化执行，提高工作效率
- **个性化定制**：支持根据业务场景定制 Agent 的行为模式和决策逻辑
- **与 LLM 深度集成**：基于 LLM 实现自然语言理解和生成能力，支持灵活的 Agent 角色定义

#### 📚 RAG 服务
Weave 的 RAG（检索增强生成）服务提供了高效的向量检索和增强生成能力，将外部知识库与 LLM 结合，提升生成内容的准确性、相关性和时效性。
- **高性能向量检索**：基于 RedisSearch 实现毫秒级向量相似度搜索，支持大规模向量库
- **多格式文档支持**：支持文本、PDF、Markdown、Word 等多种文档格式的解析、分块和向量化
- **智能文档处理**：自动进行文档分块、元数据提取和内容结构化处理，优化检索效果
- **灵活检索策略**：支持混合检索（向量+关键词）、语义检索、相似性阈值过滤等多种检索算法和参数配置
- **自定义嵌入模型**：支持切换不同的嵌入模型（如 OpenAI Embeddings、BGE 等），适应不同场景的向量表示需求
- **与 LLM 无缝集成**：检索结果自动与 LLM 生成内容结合，生成准确、权威的响应
- **知识库管理**：提供完整的知识库管理功能，支持文档添加、删除、更新和版本控制
- **检索结果可解释**：提供检索结果的相关性分数和来源信息，增强生成内容的可解释性

服务聚合的设计提升了系统功能灵活性，允许系统扩展整合各类服务和数据源，提供更强大的底层能力支持。

### 🔐 认证系统
认证系统位于分层架构的基础设施层，提供完善的身份认证和授权机制，支持多种认证方式，认证系统与插件系统紧密结合，确保插件的安全访问，同时通过分层设计实现了安全机制的统一管理
- 基于 JWT 的令牌认证
- 支持访问令牌和刷新令牌机制
- 密码哈希存储，增强安全性
- 登录历史记录，便于审计和追踪
- 基于角色的访问控制

### 🔄 中间件系统
中间件系统位于分层架构的接口层和业务层之间，支持全局中间件和插件级中间件，可用于日志记录、请求验证、性能监控等场景。中间件系统采用链式调用模式，灵活组合各种功能，体现了分层架构的请求处理优化
- 认证中间件：验证用户身份
- 限流中间件：防止API滥用
- CORS中间件：处理跨域请求
- CSRF保护中间件：防止跨站请求伪造
- 错误处理中间件：统一处理和记录错误

### 📈 监控系统
Weave 集成了完整 Prometheus + Grafana 监控系统：
- 自动采集应用运行指标
- 预置多种可视化仪表盘
- 支持自定义告警规则
- 实时监控系统健康状态和性能指标

---

## 快速开始

### 环境准备
- **Go 1.24+**
- **Docker** and **Docker Compose**
- **MySQL 8.0+**
- **PostgreSQL、Redis 7.0+、Prometheus、Grafana**（可选、扩展）

### 部署方式

#### 1. Docker Compose 部署（推荐）

1. 克隆代码库
```bash
git clone https://github.com/liaotxcn/weave.git
cd weave
```

2. 创建环境变量文件（可选但推荐）
`.env`文件，设置环境变量以增强安全性

3. 启动服务
使用Docker Compose一键启动整个服务栈：
```bash
docker-compose up -d
```
   首次启动时，Docker Compose会自动：
   - 构建Weave应用的Docker镜像
   - 创建MySQL数据库容器
   - 创建RedisSearch向量数据库容器
   - 配置Prometheus和Grafana监控系统
   - 配置网络和卷
   - 启动所有服务
   
   服务启动后，可以访问以下地址：
   - 应用后端：http://localhost:8081
   - Prometheus监控：http://localhost:9090
   - Grafana仪表盘：http://localhost:3000（默认账号密码：admin/admin）

4. 验证服务状态
查看所有服务是否正常运行：
```bash
docker-compose ps
```
正常情况下，`weave-app`、`weave-mysql`和`weave-redis`都应显示为`Up`状态。

### Docker Compose 命令

```bash
docker-compose down    // 停止服务
docker-compose logs -f weave-app   // 查看应用日志
docker-compose logs -f weave-mysql // 查看数据库日志
docker-compose logs -f weave-redis    // 查看Redis日志
docker-compose exec weave-app /bin/sh             // 进入应用容器
docker-compose exec weave-mysql mysql -u root -p  // 进入数据库容器
docker-compose exec weave-redis redis-cli    // 进入Redis容器
docker-compose up --build -d        // 重新构建并启动服务

// 清理旧容器和卷数据
docker-compose down -v 
docker system prune -f
docker-compose build --no-cache     // 重建镜像
docker-compose up --force-recreate -d   // 使用--force-recreate选项启动
```

#### 2. 本地开发环境设置

1. 克隆代码库并进入项目目录
```bash
git clone https://github.com/liaotxcn/weave.git
cd weave
```

2. 安装依赖
```bash
go mod download
```

3. 配置数据库
确保本地MySQL服务已启动，并创建数据库：
```sql
CREATE DATABASE weave;
```

4. 设置环境变量或修改`config/config.go`中的默认配置

5. 运行应用
```bash
go run main.go
```

6. 构建应用
```bash
go build
```

#### 前端构建
```bash
cd web
npm install
npm run dev
```

### 注意事项

1. **数据持久化**：
   - MySQL数据存储在`mysql-data`卷中，确保数据不会丢失
   - RedisSearch数据存储在`redis-data`卷中，确保向量索引数据不会丢失
2. **健康检查**：系统提供`/health`接口监控服务健康状态
3. **资源限制**：默认配置了CPU和内存限制，可根据实际需求在`docker-compose.yaml`中调整
4. **首次启动**：首次启动需要一些时间来构建镜像和初始化服务
5. **端口映射**：
   - 默认将容器的8081端口映射到主机的8081端口
   - 默认将容器的6379端口映射到主机的6379端口（RedisSearch）

### 扩展功能/工具库 [Weave-Toolkit](https://github.com/liaotxcn/weave-toolkit) 

---

## 🤝 贡献指南

欢迎对项目进行贡献！感谢！

1. **Fork 仓库**并克隆到本地
2. **创建分支**进行开发（`git checkout -b feature/your-feature`）
3. **提交代码**并确保通过测试
4. **创建 Pull Request** 描述您的更改
5. 等待**代码审查**并根据反馈进行修改

---

### <div align="center"> <strong>✨ 持续更新完善中... ✨</strong> </div>
