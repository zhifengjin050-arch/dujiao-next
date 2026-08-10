# 🦄 Dujiao-Next

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Vue Version](https://img.shields.io/badge/Vue-3.5-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org/)
[![Vite](https://img.shields.io/badge/Vite-7.2-646CFF?style=flat&logo=vite)](https://vitejs.dev/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-3.4-06B6D4?style=flat&logo=tailwindcss)](https://tailwindcss.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhifengjin050-arch/dujiao-next)](https://goreportcard.com/report/github.com/zhifengjin050-arch/dujiao-next)

**下一代高性能虚拟商品自动发货系统**

一站式虚拟商品交易平台，支持多渠道支付、自动发货、多端协同，为虚拟商品交易提供完整的数字化解决方案。

[功能特性](#-功能特性) •
[技术架构](#-技术架构) •
[快速开始](#-快速开始) •
[部署指南](#-部署指南) •
[API 文档](#-api-文档) •
[贡献指南](CONTRIBUTING.md) •
[更新日志](CHANGELOG.md)

</div>

---

## 📖 目录

- [项目背景](#-项目背景)
- [功能特性](#-功能特性)
- [技术架构](#-技术架构)
- [项目结构](#-项目结构)
- [快速开始](#-快速开始)
- [部署指南](#-部署指南)
- [配置说明](#-配置说明)
- [API 文档](#-api-文档)
- [支付渠道](#-支付渠道)
- [安全特性](#-安全特性)
- [常见问题](#-常见问题)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

## 🎯 项目背景

Dujiao-Next 是一个面向虚拟商品交易的现代化自动发货系统。在数字经济时代，虚拟商品（如卡密、激活码、在线服务等）的交易需求日益增长，但传统电商平台在虚拟商品处理、自动发货、多渠道支付集成等方面存在明显不足。

本项目旨在解决以下核心痛点：

- **自动化发货**：订单支付成功后自动完成商品交付，无需人工干预
- **多渠道支付**：集成支付宝、微信支付、PayPal、Stripe 等主流支付渠道
- **多端协同**：提供管理后台、用户前台等多端支持
- **上游对接**：支持作为下游站点对接上游 API，实现商品和库存同步
- **安全可靠**：完善的鉴权体系、风控机制和数据加密方案

### 技术栈优势

- **全栈 Go + Vue 3**：高性能后端与现代化前端的最佳组合
- **TypeScript 全链路**：前后端类型安全，降低运行时错误
- **Docker 容器化**：一键部署，环境一致性保障
- **多语言国际化**：内置中英文支持，可扩展更多语言
- **模块化架构**：清晰的分层设计，便于二次开发和维护

## ✨ 功能特性

### 🏪 核心业务

| 功能模块 | 描述 |
|---------|------|
| 商品管理 | 多规格商品（SKU）、分类管理、库存管理、上下架控制 |
| 订单管理 | 订单创建、支付追踪、自动发货、手动补单、订单导出 |
| 购物车 | 用户购物车、游客购物车、批量下单 |
| 卡密管理 | 卡密导入/导出、自动发货、库存扣减、已售标记 |
| 支付集成 | 多支付渠道统一管理、支付回调、手动补单 |
| 会员体系 | 会员等级、积分系统、优惠策略 |

### 👤 用户中心

| 功能模块 | 描述 |
|---------|------|
| 注册登录 | 邮箱注册、邮箱验证码登录、密码找回、记住我 |
| 个人中心 | 个人信息管理、密码修改、邮箱变更 |
| 钱包系统 | 余额充值、交易记录、多支付渠道充值 |
| 礼品卡 | 礼品卡兑换、余额充值 |
| 推广联盟 | 推广链接、佣金统计、佣金提现 |

### 🛠 管理后台

| 功能模块 | 描述 |
|---------|------|
| 数据看板 | 销售额统计、订单趋势、支付渠道分析 |
| 商品管理 | 商品 CRUD、SKU 管理、批量操作、库存预警 |
| 分类管理 | 商品分类树形管理、排序 |
| 订单管理 | 订单列表、订单详情、手动发货、订单取消/退款 |
| 用户管理 | 用户列表、登录日志、封禁管理 |
| 内容管理 | 文章发布、Banner 管理、富文本编辑器 |
| 支付配置 | 多渠道支付参数配置、费率设置、启用/禁用 |
| 系统配置 | 站点设置、邮件配置、安全策略、CORS 配置 |
| 管理员权限 | 角色管理、权限分配、操作日志 |
| 上游管理 | 上游 API 配置、商品同步、订单回调 |

### 🔌 技术亮点

| 特性 | 描述 |
|------|------|
| 多语言支持 | 前后端国际化（i18n），支持中文简体/繁体/英文 |
| 验证码系统 | 图片验证码、邮箱验证码双重验证 |
| 限流保护 | 登录限流、API 限流、IP 黑名单 |
| 上游 API | 完整的上下游对接协议，支持商品/订单同步 |
| 异步任务 | 基于 Redis 的异步队列，支持上游同步 |
| 图片处理 | 图片上传、尺寸限制、格式校验 |

## 🏗 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                       Nginx (反向代理)                        │
├──────────────┬──────────────────┬────────────────────────────┤
│   Admin SPA  │    User SPA      │      API Service (Go)      │
│  (Vue 3 + TS)│  (Vue 3 + TS)    │   (Gin + GORM + Redis)    │
│   Port 5174  │    Port 5173     │        Port 8080           │
├──────────────┴──────────────────┴────────────────────────────┤
│                      数据层                                   │
│   ┌──────────┐  ┌──────────┐  ┌──────────────────────────┐  │
│   │ SQLite / │  │  Redis   │  │   File Storage (uploads)  │  │
│   │PostgreSQL│  │ (缓存/队列)│  │                          │  │
│   └──────────┘  └──────────┘  └──────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    外部服务集成                                │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│   │ 支付宝   │  │ 微信支付  │  │  PayPal  │  │  Stripe  │  │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│   ┌──────────┐  ┌──────────┐  ┌──────────────────────────┐  │
│   │ 易支付   │  │ Telegram │  │  上游 API / 下游回调     │  │
│   └──────────┘  └──────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| **后端框架** | Go 1.25 + Gin | 高性能 HTTP 框架，RESTful API |
| **ORM** | GORM | 优雅的 Go ORM 框架 |
| **数据库** | SQLite / PostgreSQL | 开发用 SQLite，生产推荐 PostgreSQL |
| **缓存** | Redis | 会话缓存、限流计数、异步队列 |
| **前端框架** | Vue 3 + TypeScript | Composition API，类型安全 |
| **构建工具** | Vite 7 | 极速开发体验 |
| **UI 框架** | Tailwind CSS + Reka UI | 原子化 CSS + 无头 UI 组件 |
| **状态管理** | Pinia | Vue 3 官方状态管理 |
| **国际化** | vue-i18n / go-i18n | 前后端完整国际化 |
| **富文本** | TipTap | 基于 ProseMirror 的现代编辑器 |

## 📁 项目结构

```
dujiao-next/
├── api/                          # Go 后端服务
│   ├── cmd/server/               # 应用入口
│   ├── internal/
│   │   ├── adgateway/            # 广告网关
│   │   ├── app/                  # 应用初始化
│   │   ├── authz/                # 权限鉴权
│   │   ├── cache/                # Redis 缓存
│   │   ├── config/               # 配置管理
│   │   ├── constants/            # 常量定义
│   │   ├── crypto/               # 加密工具
│   │   ├── dto/                  # 数据传输对象
│   │   ├── http/
│   │   │   ├── handlers/         # HTTP 处理器
│   │   │   │   ├── admin/        # 管理端接口
│   │   │   │   ├── channel/      # 渠道接口
│   │   │   │   ├── public/       # 公开接口
│   │   │   │   └── upstream/     # 上游接口
│   │   │   ├── middleware/       # 中间件
│   │   │   └── response/         # 响应封装
│   │   ├── i18n/                 # 国际化
│   │   ├── logger/               # 日志系统
│   │   ├── models/               # 数据模型
│   │   ├── payment/              # 支付核心
│   │   │   └── provider/         # 支付渠道实现
│   │   │       ├── alipay/       # 支付宝
│   │   │       ├── epay/         # 易支付
│   │   │       ├── epusdt/       # USDT 支付
│   │   │       ├── paypal/       # PayPal
│   │   │       ├── stripe/       # Stripe
│   │   │       ├── wechatpay/    # 微信支付
│   │   │       └── xunhupay/     # 虎皮椒支付
│   │   ├── queue/                # 异步任务队列
│   │   ├── repository/           # 数据访问层
│   │   ├── router/               # 路由定义
│   │   ├── service/              # 业务逻辑层
│   │   ├── upstream/             # 上游对接
│   │   ├── version/              # 版本信息
│   │   └── worker/               # 后台任务
│   ├── config.yml.example        # 配置示例
│   ├── .env.example              # 环境变量示例
│   ├── Dockerfile                # Docker 构建
│   ├── go.mod                    # Go 依赖
│   └── go.sum
├── admin/                        # Vue 管理后台
│   ├── src/
│   │   ├── api/                  # API 请求层
│   │   ├── components/           # 通用组件
│   │   │   └── ui/               # UI 基础组件
│   │   ├── composables/          # 组合式函数
│   │   ├── constants/            # 常量
│   │   ├── i18n/                 # 国际化
│   │   ├── layouts/              # 布局组件
│   │   ├── lib/                  # 工具库
│   │   ├── router/               # 路由配置
│   │   ├── stores/               # Pinia 状态
│   │   ├── utils/                # 工具函数
│   │   └── views/                # 页面视图
│   │       └── admin/            # 管理页面
│   ├── .env.example              # 环境变量示例
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   └── package.json
├── user/                         # Vue 用户前台
│   ├── src/
│   │   ├── api/                  # API 请求层
│   │   ├── components/           # 通用组件
│   │   │   └── captcha/          # 验证码组件
│   │   ├── composables/          # 组合式函数
│   │   ├── constants/            # 常量
│   │   ├── i18n/                 # 国际化
│   │   ├── router/               # 路由配置
│   │   ├── stores/               # Pinia 状态
│   │   ├── types/                # TypeScript 类型
│   │   ├── utils/                # 工具函数
│   │   └── views/                # 页面视图
│   │       ├── auth/             # 认证页面
│   │       ├── checkout/         # 结算页面
│   │       ├── payment/          # 支付页面
│   │       ├── personal/         # 个人中心
│   │       ├── product/          # 商品页面
│   │       ├── security/         # 安全设置
│   │       └── wallet/           # 钱包页面
│   ├── .env.example              # 环境变量示例
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
├── docker-compose.yml            # Docker 编排
├── .gitignore
├── LICENSE                       # MIT 许可证
├── README.md                     # 项目说明
├── CONTRIBUTING.md               # 贡献指南
├── CODE_OF_CONDUCT.md            # 行为准则
└── CHANGELOG.md                  # 更新日志
```

## 📸 界面预览

> 以下为系统主要界面截图，展示核心功能模块。

| 界面 | 说明 |
|------|------|
| 🖥️ **管理后台 - 数据看板** | 销售统计、订单趋势、支付渠道分析 |
| 📦 **管理后台 - 商品管理** | 多规格商品（SKU）管理、库存管理 |
| 📋 **管理后台 - 订单管理** | 订单列表、详情、手动发货、导出 |
| 👤 **用户前台 - 商品浏览** | 商品列表、分类筛选、搜索 |
| 🛒 **用户前台 - 购物车** | 购物车管理、批量下单 |
| 💰 **用户前台 - 支付结算** | 多支付渠道选择、订单确认 |

> *提示：请将实际截图放入 `docs/screenshots/` 目录，并更新此表格中的图片链接。*

## 🚀 快速开始

### 环境要求

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| Go | >= 1.24 | 后端编译运行 |
| Node.js | >= 18 | 前端构建 |
| Redis | >= 6.0 | 缓存和队列（可选，开发环境可关闭） |
| SQLite | 内置 | 默认数据库（开发环境） |
| PostgreSQL | >= 14 | 生产环境推荐 |

### 本地开发

#### 1. 克隆项目

```bash
git clone https://github.com/zhifengjin050-arch/dujiao-next.git
cd dujiao-next
```

#### 2. 启动后端 API

```bash
cd api

# 复制配置文件
cp config.yml.example config.yml

# 编辑 config.yml，修改必要配置项
# 特别是 app.secret_key、jwt.secret、user_jwt.secret

# 安装依赖
go mod tidy

# 启动服务（默认端口 8080）
go run cmd/server/main.go
```

#### 3. 启动管理后台

```bash
cd admin

# 复制环境变量文件
cp .env.example .env

# 安装依赖
npm install

# 启动开发服务器（默认端口 5174）
npm run dev
```

#### 4. 启动用户前台

```bash
cd user

# 复制环境变量文件
cp .env.example .env

# 安装依赖
npm install

# 启动开发服务器（默认端口 5173）
npm run dev
```

#### 5. 访问系统

| 端 | 地址 | 说明 |
|----|------|------|
| API 服务 | http://localhost:8080 | 后端 API，健康检查 `/health` |
| 管理后台 | http://localhost:5174 | 管理员登录后台 |
| 用户前台 | http://localhost:5173 | 用户商城前台 |

## 📦 部署指南

### Docker 部署（推荐）

```bash
# 克隆项目
git clone https://github.com/zhifengjin050-arch/dujiao-next.git
cd dujiao-next

# 修改 docker-compose.yml 中的 JWT_SECRET
# 生成随机密钥: openssl rand -hex 32

# 一键启动
docker-compose up -d
```

### 生产环境手动部署

#### 1. 构建后端

```bash
cd api

# Linux 交叉编译
GOOS=linux GOARCH=amd64 go build -o dujiao-api cmd/server/main.go

# 或使用 goreleaser
goreleaser build --snapshot --clean
```

#### 2. 构建前端

```bash
# 构建管理后台
cd admin && npm run build

# 构建用户前台
cd user && npm run build
```

#### 3. Nginx 配置示例

```nginx
# API 服务
server {
    listen 80;
    server_name api.your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    client_max_body_size 20m;
}

# 管理后台
server {
    listen 80;
    server_name admin.your-domain.com;

    root /path/to/admin/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # 静态资源缓存
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}

# 用户前台
server {
    listen 80;
    server_name shop.your-domain.com;

    root /path/to/user/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

#### 4. 生产配置检查清单

```bash
# 复制配置模板
cp api/config.yml.example api/config.yml

# 编辑配置文件，修改以下关键项：
# - app.secret_key: 生成随机密钥（openssl rand -hex 32）
# - jwt.secret: 生成随机密钥
# - user_jwt.secret: 生成随机密钥
# - database.dsn: 生产数据库连接
# - server.mode: 改为 release
# - redis.enabled: 开启
# - queue.enabled: 开启
```

## ⚙️ 配置说明

项目使用 YAML 配置文件，完整配置项参见 [api/config.yml.example](api/config.yml.example)。

### 关键配置项

| 配置项 | 说明 | 注意事项 |
|--------|------|---------|
| `app.secret_key` | AES-256 加密密钥 | **必须修改**，用于加密敏感数据 |
| `jwt.secret` | 管理员 JWT 签名密钥 | **必须修改**，建议 32+ 字符 |
| `user_jwt.secret` | 用户 JWT 签名密钥 | **必须修改**，建议 32+ 字符 |
| `database.driver` | 数据库驱动 | `sqlite` 或 `postgres` |
| `database.dsn` | 数据库连接字符串 | 生产环境使用 PostgreSQL |
| `server.mode` | 运行模式 | `debug` / `release` |
| `redis.enabled` | Redis 开关 | 生产环境建议开启 |
| `queue.enabled` | 异步队列开关 | 依赖 Redis |
| `bootstrap.*` | 初始化管理员 | 首次运行自动创建管理员账号 |

### 安全配置

```yaml
security:
  login_rate_limit:
    window_seconds: 300    # 限流时间窗口（秒）
    max_attempts: 5        # 最大尝试次数
    block_seconds: 900     # 封禁时长（秒）
  password_policy:
    min_length: 8          # 密码最小长度
    require_upper: true    # 必须包含大写字母
    require_lower: true    # 必须包含小写字母
    require_number: true   # 必须包含数字
    require_special: false # 是否要求特殊字符
```

### 环境变量

除了 YAML 配置文件，系统也支持通过环境变量配置（Docker 部署推荐）：

```bash
# 复制环境变量示例
cp api/.env.example api/.env
cp admin/.env.example admin/.env
cp user/.env.example user/.env
```

## 📡 API 文档

### API 概述

系统提供 RESTful API，按访问权限分为以下几组：

| 路由前缀 | 说明 | 鉴权方式 |
|---------|------|---------|
| `/api/v1/public/*` | 公开接口 | 无需鉴权 |
| `/api/v1/guest/*` | 游客接口 | 无需鉴权 |
| `/api/v1/auth/*` | 认证接口 | 登录限流 |
| `/api/v1/me/*` | 用户接口 | User JWT |
| `/api/v1/admin/*` | 管理接口 | Admin JWT |
| `/api/v1/upstream/*` | 上游 API | API Key |
| `/api/v1/channel/*` | 渠道接口 | Channel Token |

### 主要端点

#### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/public/config` | 获取站点配置 |
| GET | `/api/v1/public/products` | 商品列表 |
| GET | `/api/v1/public/products/:slug` | 商品详情 |
| GET | `/api/v1/public/categories` | 分类列表 |
| GET | `/api/v1/public/posts` | 文章列表 |
| GET | `/api/v1/public/captcha/image` | 图片验证码 |

#### 用户接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| GET | `/api/v1/me` | 获取当前用户 |
| GET | `/api/v1/cart` | 购物车 |
| POST | `/api/v1/orders` | 创建订单 |
| GET | `/api/v1/orders` | 订单列表 |
| POST | `/api/v1/payments` | 创建支付 |
| POST | `/api/v1/payments/:id/capture` | 支付确认 |
| GET | `/api/v1/wallet` | 钱包信息 |

### 健康检查

```bash
curl http://localhost:8080/health
```

## 💳 支付渠道

| 支付渠道 | 提供者 | 支持状态 | 说明 |
|---------|--------|---------|------|
| 支付宝 | `alipay` | ✅ 支持 | 支持网页/扫码支付 |
| 微信支付 | `wechatpay` | ✅ 支持 | 支持 JSAPI/Native |
| PayPal | `paypal` | ✅ 支持 | 国际支付 |
| Stripe | `stripe` | ✅ 支持 | 信用卡支付 |
| 易支付 | `epay` | ✅ 支持 | 第三方聚合支付 |
| USDT | `epusdt` | ✅ 支持 | 加密货币支付 |
| 虎皮椒 | `xunhupay` | ✅ 支持 | 微信/支付宝 |
| TokenPay | `tokenpay` | ✅ 支持 | 多链支付 |
| OKPay | `okpay` | ✅ 支持 | 虚拟货币支付 |

## 🔒 安全特性

- **JWT 双令牌体系**：管理员和用户独立的 JWT 密钥和过期策略
- **登录限流**：基于 IP + 邮箱的多维度登录频率限制，防止暴力破解
- **密码策略**：可配置的密码复杂度要求（长度、大小写、数字、特殊字符）
- **AES-256 加密**：敏感数据使用 AES-256-GCM 加密存储
- **CORS 白名单**：可配置的跨域来源控制，防止 CSRF 攻击
- **API 限流**：上游 API 调用频率限制，防止滥用
- **验证码保护**：图片验证码 + 邮箱验证码双重保护
- **敏感信息分离**：所有密钥和凭证通过配置文件管理，不硬编码
- **配置文件保护**：`.gitignore` 排除 `config.yml` 和 `.env` 文件，仅提供 `.example` 模板

## ❓ 常见问题

<details>
<summary><b>1. 启动时提示 "config.yml not found"</b></summary>

需要先复制配置模板：
```bash
cp api/config.yml.example api/config.yml
```
然后编辑 `config.yml` 修改必要配置项。
</details>

<details>
<summary><b>2. 如何从 SQLite 迁移到 PostgreSQL？</b></summary>

1. 修改 `config.yml` 中的 `database.driver` 为 `postgres`
2. 修改 `database.dsn` 为 PostgreSQL 连接字符串
3. 使用数据库迁移工具导出数据
4. 重启服务，GORM 会自动创建表结构
</details>

<details>
<summary><b>3. 前端页面无法连接后端 API？</b></summary>

检查前端 `.env` 文件中的 `VITE_API_BASE_URL` 是否正确指向后端地址。开发环境默认为 `http://localhost:8080`。
</details>

<details>
<summary><b>4. Docker 部署后如何修改配置？</b></summary>

通过环境变量覆盖配置，在 `docker-compose.yml` 的 `environment` 中添加对应变量。例如：
```yaml
environment:
  - JWT_SECRET=your-new-secret
  - DB_TYPE=postgres
  - DB_DNS=host=postgres user=dujiao password=secret dbname=dujiao
```
</details>

<details>
<summary><b>5. 如何添加新的支付渠道？</b></summary>

1. 在 `api/internal/payment/provider/` 下创建新的支付提供者
2. 实现 `PaymentProvider` 接口
3. 在 `api/internal/payment/` 中注册新的提供者
4. 在管理后台添加支付配置
</details>

## 🤝 贡献指南

我们欢迎任何形式的贡献！请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详细的贡献流程和规范。

### 快速参与

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feat/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feat/amazing-feature`)
5. 创建 Pull Request

### 行为准则

本项目遵循 [Contributor Covenant 行为准则](CODE_OF_CONDUCT.md)。参与即表示您同意遵守其条款。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。

Copyright (c) 2026 Dujiao-Next

---

<div align="center">

**Dujiao-Next** — 让虚拟商品交易更简单 🦄

⭐ 如果这个项目对你有帮助，请给一个 Star！

[报告问题](https://github.com/zhifengjin050-arch/dujiao-next/issues) •
[请求功能](https://github.com/zhifengjin050-arch/dujiao-next/issues) •
[参与讨论](https://github.com/zhifengjin050-arch/dujiao-next/discussions)

</div>