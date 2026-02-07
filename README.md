# A highly efficient, secure, and stable application development platform with excellent performance, easy scalability, and deep integration of AI capabilities such as LLM, AI Chat, RAG, and Agents.

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/Microkernel-Layered-6BA539?style=for-the-badge" alt="Architecture">
  <img src="https://img.shields.io/badge/LLM--Agent--MCP-74AA9C?style=for-the-badge&logo=brain&logoColor=white" alt="LLM-Agent-MCP">
  <img src="https://img.shields.io/badge/AIChat--RAG-FF6F00?style=for-the-badge&logo=ai&logoColor=white" alt="AIChat-RAG">
  <img src="https://img.shields.io/badge/Cloud_Native-3371E3?style=for-the-badge&logo=Docker&logoColor=white" alt="Cloud Native">
</div>

## 📋 Project Introduction

Weave (meaning "to weave") – from a simple thread to a complex tapestry, weaving is the creative process from simplicity to complexity. A high-performance, high-efficiency, easily extensible **empowerment tool/AI application development platform** built with Golang, designed for efficiently constructing stable and reliable intelligent applications. It employs a microkernel + layered architecture, allowing developers to develop efficiently and integrate/manage various tools/services with ease, while maintaining high system performance and scalability.

<img width="660" height="265" alt="image" src="https://github.com/user-attachments/assets/1c4b8e34-aa4b-496a-8a27-ec7f212e0cc7" />

Weave's core strength lies in its stable and reliable **AI feature development stack + plugin/tool system**, deeply integrating core AI capabilities such as LLM (Large Language Models), AIChat (Intelligent Chat), Agent (Intelligent Agent), RAG (Retrieval-Augmented Generation), and more. It provides a unified development framework and standardized interfaces, enabling developers to rapidly build powerful intelligent applications without needing to focus on underlying complex implementations.

Primary application scenarios include:
- Tool/Application Development Aggregation
- Data/Service Flow Middle Platform
- API Gateway and Service Orchestration
- AI Application Development - Building LLM, AIChat, Agent, and RAG intelligent applications
- Automated Workflows - Complex task automation combining ETL, DevOps, and Agent

---

## 🏗️ Overall Architecture

<img width="1562" height="545" alt="image" src="https://github.com/user-attachments/assets/e56f5d3b-b9e4-417a-992e-2997991aa2ad" />

Weave adopts a microkernel + layered architectural design pattern, fully leveraging the advantages of both to ensure system availability, achieving high flexibility, scalability, and excellent performance.

### Integration of Microkernel and Layered Architecture

Weave incorporates the design philosophy of layered architecture on top of a microkernel architecture, forming a complete, efficient, and flexible architectural system:

- Microkernel Architecture (Plugin System): Provides plugin management, lifecycle control, and inter-plugin communication mechanisms.
- Layered Architecture (Core System): Separates core functionalities by concerns, forming a clear hierarchical structure.

### Microkernel Architecture Components

- Core Kernel: Provides fundamental runtime environment, plugin management, configuration management, logging services, security mechanisms, and other basic functionalities.
- Plugin System: The plugin manager handles plugin registration, lifecycle management, dependency resolution, and conflict detection.
- Extension Plugins: Integrated into the core system through plugin interfaces, implementing various business functionalities.

### Layered Architecture Components

- Interface Layer: Handles HTTP requests, including routing management and controllers.
- Business Layer: Contains core business logic and the plugin system.
- Data Layer: Responsible for data storage and access.
- Infrastructure Layer: Provides services such as logging, configuration, and security.

### Architectural Characteristics

- Loosely Coupled Design: The core system and plugins communicate through well-defined interfaces, reducing inter-module dependencies.
- Hot-Plug Capability: Plugins can be dynamically loaded and unloaded at runtime without system restart.
- Feature Isolation: Each plugin independently encapsulates functionality, with its own namespace and route prefix.
- Dependency & Conflict Management: Built-in dependency resolution and conflict detection mechanisms ensure harmonious coexistence among plugins.
- Unified Interface: All plugins implement the same `Plugin` interface, standardizing the development process.
- Extensibility: System functionalities can be extended on-demand without modifying kernel code.
- Clear Hierarchy: The core system uses layered design for rational code organization, making it easy to maintain and extend.
- High Performance: The layered design optimizes request processing flow, improving system response speed.

The system's core is an efficient and stable plugin mechanism and service aggregation, allowing functional modules to be independently developed and deployed as plugins/services, while interacting through unified interfaces. The overall architectural design emphasizes modularity, scalability, and high performance.

---

## 🌟 Project Features

### 🏗️ Microkernel + Layered Architecture
- **Stable Core & Clear Hierarchy**: The core system remains minimal, and the layered design ensures rational code organization, making it easy to maintain and extend.
- **Flexible Functional Extension**: Extend system functionalities on-demand through the plugin mechanism without modifying kernel code.
- **Low Coupling, High Cohesion**: System components are loosely coupled, facilitating maintenance and upgrades.
- **Hot-Plug Capability**: Plugins can be dynamically loaded and unloaded at runtime without restarting the system.
- **Feature Isolation & Unified Management**: Each plugin independently encapsulates functionality, with its own namespace and route prefix, while core services are uniformly managed through the layered architecture.
- **Dependency & Conflict Management**: Built-in dependency resolution and conflict detection mechanisms ensure harmonious plugin coexistence.
- **Unified Interface**: All plugins implement the same `Plugin` interface, standardizing the development process.
- **High Performance**: The layered design optimizes the request processing flow, improving system response speed.

### 🚀 High Performance/Efficiency
- Built on the Gin framework, offering high performance and strong concurrency capabilities.
- Optimized database connection pool supporting high-concurrency access.
- Modular architectural design with a clear code structure, making it easy to maintain and extend.
- Supports environment variable overrides for easy configuration across different environments.
- Efficient routing management supporting dynamic routes and parameter binding.
- The layered architecture optimizes request processing flow, enhancing system response speed.

### 🔌 Plug-in Easy Extensibility
- Unified plugin interface design supporting hot-plugging.
- Plugin manager uniformly registers, manages, and executes plugins.
- Plugins can independently register routes with their own namespace.
- Plugin dependency and conflict detection mechanisms.
- Scaffolding tool for conveniently generating plugin framework code.
- Example plugins demonstrating the complete plugin development process.

### 🧠 Deep AI Integration
Weave provides a complete AI development functional stack, deeply integrating core AI technologies such as LLM, AIChat, Agent, and RAG, offering developers a one-stop intelligent application development experience.

- **Complete AI Functional Stack**: Integrates core AI capabilities like LLM (Large Language Models), AIChat (Intelligent Chat), Agent (Intelligent Agent), and RAG (Retrieval-Augmented Generation), covering the entire process from basic model invocation to complex intelligent application development.
- **Multi-Model Support**: Compatible with mainstream LLM platforms like OpenAI, Ollama, ModelScope, etc., supporting dynamic switching of models and platforms to meet different scenario needs.
- **Comprehensive AIChat Services**: Multi-platform, multi-model, multimodal support, streaming response, context optimization, summary extraction, conversation management, and other features.
- **Intelligent Agent Framework**: Provides a complete Agent development framework supporting tool calling, task planning, complex workflow automation, and multi-turn conversation management.
- **Efficient RAG System**: Implements high-performance vector search based on RedisSearch, supports multi-format document processing (text, PDF, Markdown, etc.), improving the accuracy and relevance of generated content.
- **Flexible Embedding Models**: Supports custom embedding models and retrieval parameters to adapt to different business scenarios' vector representation needs.
- **Seamless Architectural Integration**: AI capabilities can be used as service aggregates or plugins, facilitating rapid construction and extension of intelligent applications.

### 🔒 Secure & Reliable
- JWT-based authentication and authorization system.
- Comprehensive CSRF protection mechanism.
- Rate-limiting middleware based on the token bucket algorithm.
- Password hashing storage and verification.
- Detailed login history records.
- Unified error handling middleware.
- HTTPS support (can be enabled in configuration).
- The layered architecture encapsulates security mechanisms uniformly in the infrastructure layer for easy unified management and maintenance.

### 📊 Observability
- Integrated structured logging system (zap).
- Health check endpoint for monitoring system status.
- Detailed request/response logging.
- Support for custom monitoring metrics.
- The layered architecture independently encapsulates monitoring functionalities, ensuring the observability of each layer's operational state.
- Integrated Prometheus and Grafana monitoring system providing visual dashboards.
- Supports custom alert rule configuration.

### 🚀 Developer-Friendly
- Complete plugin development documentation and examples.
- Plugin scaffolding tool for quickly generating plugin templates.
- Supports local development and Docker deployment.
- Project structure and code standards.
- Clear modules and standardized interfaces, facilitating development and maintenance.

---

## 📂 Project Structure

Weave adopts a microkernel + layered architecture, and its project structure clearly reflects this design philosophy. The core system is organized in layers, while functional extensions are achieved through the plugin mechanism/service aggregation.

```bash
├── Dockerfile           # Docker build file
├── Makefile             # Build scripts
├── config/              # Configuration management
├── controllers/         # API controllers [Interface Layer]
├── docker-compose.yaml  # Docker Compose configuration
├── docs/                # Project documentation
├── main.go
├── middleware/          # Middleware
├── models/              # Data models [Data Layer]
├── pkg/                 # Common packages [Infrastructure Layer]
├── plugins/             # Plugin system [Core of Microkernel Architecture]
│ ├── core/              # Core plugin functionalities
│ ├── examples/          # Example plugins
│ ├── features/          # Feature plugins (extensible)
│ ├── init.go            # Plugin initialization
│ ├── loader/            # Plugin loader
│ ├── templates/         # Plugin templates
│ └── watcher/           # Plugin watcher
├── routers/             # Route definition and registration
├── services/            # Service aggregation
│ ├── aichat/            # aichat service
│   ├── chat_web/
│   ├── cmd/               # Entry point
│   ├── internal/          # Core implementation
│   ├── pkg/
│ ├── rag/               # RAG service
│ ├── email/             # Email service
│ └── extended...        # Extensible services
├── test/                # Unit/Integration tests
├── tools/               # Development tools
├── utils/               # Utility functions
└── web/                 # Frontend code
```

---

## 🧩 Core Components

### 🔌 Plugin System - The Core Implementation of Microkernel Architecture
The plugin system is a vital component of Weave, responsible for plugin registration, loading, unloading, and lifecycle management. It implements a complete plugin mechanism, enabling the system to extend functionalities in the form of plugins. In the microkernel + layered architecture, the plugin system connects the core kernel with various business extensions.

- **Complete Lifecycle Management**: Full lifecycle management from plugin initialization, registration, activation to shutdown.
- **Automatic Dependency Resolution**: Automatically resolves dependencies between plugins via the `GetDependencies()` method.
- **Conflict Detection Mechanism**: Avoids functional conflicts between plugins via the `GetConflicts()` method.
- **Automatic Route Registration**: Supports two route registration methods, with the recommended `GetRoutes()` method particularly aligned with microkernel architectural design principles.
- **Namespace Isolation**: Each plugin has an independent namespace, avoiding resource conflicts.
- **Unified Middleware Management**: Supports both global and plugin-level middleware configuration.

#### Comparison of Two Route Registration Methods
| Feature | `GetRoutes` Method (Recommended) | `RegisterRoutes` Method (Kept for Compatibility) |
|------|-----------------------|-----------------------------------|
| Route Definition | Uses an array of `Route` structs | Directly operates on the `gin.Engine` object |
| Metadata Support | ✅ Full support | ❌ Not supported |
| Automatic Route Groups | ✅ Automatically created | ❌ Manual creation required |
| Middleware Management | ✅ Supports global and route-level | ❌ Manual addition required |
| Documentation Generation | ✅ Supports automatic API documentation generation | ❌ Not supported |

The plugin manager handles the entire lifecycle of plugins, including registration, deregistration, querying, and executing plugin functionalities.

### 🧩 Service Aggregation
Service aggregation is a significant extension capability of Weave based on the microkernel + layered architecture, providing mechanisms for unified management and invocation of various services, data sources, and functionalities. It enhances AI service aggregation capabilities, offering a complete LLM, AIChat, Agent, and RAG service system.

#### 🧠 LLM/AIChat Service
Weave's LLM service provides unified large language model access and management capabilities, allowing developers to easily invoke and switch between different platform models without needing to focus on underlying implementation details.
- **Multi-Platform Compatibility, Dynamic Model Switching**: Supports mainstream LLM platforms like OpenAI, Ollama, ModelScope, etc., allowing quick switching between different LLM models and platforms.
- **Standardized Interface**: Provides unified API interfaces, easy to develop and maintain.
- **Streaming Response Support**: Supports real-time streaming responses, enhancing user interaction experience.
- **Context Optimization**: Optimizes context preservation and management, improving conversation coherence.
- **Summary Extraction**: Automatically extracts conversation summaries for quick access to key information.
- **Multimodal Support**: Some models support multimodal input/output (text, images, etc.).
- **Model Caching Mechanism**: Built-in caching mechanism reduces duplicate requests, improving performance and lowering costs.

#### 🤖 Agent Service
Weave's Agent service provides a complete intelligent agent development and runtime framework, supporting tool calling, task planning, memory management, and complex workflow automation, enabling developers to quickly build intelligent applications with autonomous decision-making capabilities.
- **Tool Calling Capability**: Supports Agents calling various internal tools and external services, expanding the boundaries of intelligent capabilities.
- **Intelligent Task Planning**: Possesses automatic task decomposition, subtask planning, and execution path optimization capabilities.
- **Flexible Memory Management**: Supports short-term memory (session context) and long-term memory (knowledge base) management, improving decision-making continuity.
- **Complex Multi-turn Dialogue**: Supports complex multi-turn dialogue interactions, understanding and maintaining dialogue context.
- **Workflow Automation**: Enables the automation of complex business processes, improving work efficiency.
- **Personalization Customization**: Supports customizing Agent behavior patterns and decision logic based on business scenarios.
- **Deep Integration with LLM**: Implements natural language understanding and generation capabilities based on LLM, supporting flexible Agent role definition.

#### 📚 RAG Service
Weave's RAG (Retrieval-Augmented Generation) service provides efficient vector retrieval and augmented generation capabilities, combining external knowledge bases with LLMs to improve the accuracy, relevance, and timeliness of generated content.
- **High-Performance Vector Retrieval**: Implements millisecond-level vector similarity search based on RedisSearch, supporting large-scale vector databases.
- **Multi-Format Document Support**: Supports parsing, chunking, and vectorization of various document formats like text, PDF, Markdown, Word, etc.
- **Intelligent Document Processing**: Automatically performs document chunking, metadata extraction, and content structuring to optimize retrieval effectiveness.
- **Flexible Retrieval Strategies**: Supports hybrid retrieval (vector + keyword), semantic retrieval, similarity threshold filtering, and various other retrieval algorithms and parameter configurations.
- **Custom Embedding Models**: Supports switching different embedding models (e.g., OpenAI Embeddings, BGE, etc.) to adapt to different scenario needs for vector representation.
- **Seamless Integration with LLM**: Automatically combines retrieval results with LLM-generated content to produce accurate, authoritative responses.
- **Knowledge Base Management**: Provides complete knowledge base management functionalities, supporting document addition, deletion, update, and version control.
- **Explainable Retrieval Results**: Provides relevance scores and source information for retrieval results, enhancing the explainability of generated content.

The design of service aggregation enhances system flexibility, allowing the integration of various services and data sources to provide more powerful underlying capability support.

### 🔐 Authentication System
Located in the infrastructure layer of the layered architecture, the authentication system provides a comprehensive identity verification and authorization mechanism, supporting multiple authentication methods. It is tightly integrated with the plugin system to ensure secure access to plugins, while achieving unified management of security mechanisms through layered design.
- JWT-based token authentication.
- Supports access token and refresh token mechanisms.
- Password hashing storage for enhanced security.
- Login history records for easy auditing and tracking.
- Role-based access control.

### 🔄 Middleware System
The middleware system is located between the interface layer and the business layer in the layered architecture, supporting both global middleware and plugin-level middleware. It can be used for scenarios like logging, request validation, and performance monitoring. The middleware system uses a chain-of-responsibility pattern, flexibly combining various functionalities, reflecting the request processing optimization of the layered architecture.
- Authentication Middleware: Validates user identity.
- Rate Limiting Middleware: Prevents API abuse.
- CORS Middleware: Handles cross-origin requests.
- CSRF Protection Middleware: Prevents cross-site request forgery.
- Error Handling Middleware: Unifies error processing and logging.

### 📈 Monitoring System
Weave integrates a complete Prometheus + Grafana monitoring system:
- Automatically collects application runtime metrics.
- Pre-configured with various visual dashboards.
- Supports custom alert rules.
- Real-time monitoring of system health status and performance metrics.

---

## Quick Start

### Environment Preparation
- **Go 1.24+**
- **Docker** and **Docker Compose**
- **MySQL 8.0+**
- **PostgreSQL, Redis 7.0+, Prometheus, Grafana** (optional, for extension)

### Deployment Methods

#### 1. Docker Compose Deployment (Recommended)

1. Clone the repository
```bash
git clone https://github.com/liaotxcn/weave.git
cd weave
```

2. Create an environment variable file (optional but recommended)
Create a .env file to set environment variables for enhanced security.

3. Start the services
Use Docker Compose to start the entire service stack with one command:
```bash
docker-compose up -d
```
    On the first run, Docker Compose will automatically:
    - Build the Docker image for the Weave application.
    - Create the MySQL database container.
    - Create the RedisSearch vector database container.
    - Configure the Prometheus and Grafana monitoring system.
    - Configure networks and volumes.
    - Start all services.

    After services start, you can access:
    - Application Backend: http://localhost:8081
    - Prometheus Monitoring: http://localhost:9090    
    - Grafana Dashboard: http://localhost:3000 (default credentials: admin/admin)

4. Verify service status
Check if all services are running normally:
```bash
docker-compose ps
```
Normally, weave-app, weave-mysql, and weave-redis should all show as Up.

### Docker Compose Commands

```bash
docker-compose down    // Stop services
docker-compose logs -f weave-app   // View application logs
docker-compose logs -f weave-mysql // View database logs
docker-compose logs -f weave-redis // View Redis logs
docker-compose exec weave-app /bin/sh             // Enter the application container
docker-compose exec weave-mysql mysql -u root -p  // Enter the database container
docker-compose exec weave-redis redis-cli         // Enter the Redis container
docker-compose up --build -d        // Rebuild and start services

// Clean up old containers and volume data
docker-compose down -v
docker system prune -f
docker-compose build --no-cache     // Rebuild images
docker-compose up --force-recreate -d   // Start with the --force-recreate option
```

#### 2. Local Development Environment Setup

1. **Clone the repository and enter the project directory**
```bash
git clone https://github.com/liaotxcn/weave.git
cd weave
```

2. **Install dependencies**
```bash
go mod download
```

3. **Configure the database**
Ensure the local MySQL service is started and create the database:
```sql
CREATE DATABASE weave;
```

4. **Set environment variables or modify the default configuration in `config/config.go`**

5. **Run the application**
```bash
go run main.go
```

6. **Build the application**
```bash
go build
```

#### Frontend Build
```bash
cd web
npm install
npm run dev
```

### Important Notes

1. **Data Persistence**:
   - MySQL data is stored in the `mysql-data` volume to ensure data is not lost
   - RedisSearch data is stored in the `redis-data` volume to ensure vector index data is not lost

2. **Health Checks**: The system provides a `/health` endpoint to monitor service health status

3. **Resource Limits**: Default CPU and memory limits are configured, which can be adjusted in `docker-compose.yaml` based on actual requirements

4. **First Startup**: The first startup may take some time to build images and initialize services

5. **Port Mapping**:
   - By default, the container's port 8081 is mapped to the host's port 8081
   - By default, the container's port 6379 is mapped to the host's port 6379 (RedisSearch)

### Extension Functionality/Toolkit [Weave-Toolkit](https://github.com/liaotxcn/weave-toolkit)

---

## 🤝 Contribution Guidelines

Contributions to the project are welcome! Thank you!

1. **Fork the repository** and clone it locally
2. **Create a branch** for development (`git checkout -b feature/your-feature`)
3. **Commit your code** and ensure it passes tests
4. **Create a Pull Request** describing your changes
5. Wait for **code review** and make modifications based on feedback

---

### <div align="center"> <strong>✨ Continuously updating and improving... ✨</strong> </div>
