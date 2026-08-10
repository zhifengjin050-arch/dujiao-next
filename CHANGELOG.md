# Changelog

所有值得注意的项目变更都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added
- 完整的 Go 后端 API 服务，基于 Gin + GORM 框架
- Vue 3 + TypeScript 管理后台，支持多语言国际化
- Vue 3 + TypeScript 用户前台商城
- 多规格商品（SKU）管理系统
- 卡密自动发货系统
- 多支付渠道集成（支付宝、微信支付、PayPal、Stripe、易支付、USDT、虎皮椒、TokenPay、OKPay）
- 用户注册/登录系统，支持邮箱验证码和 Telegram OAuth
- 购物车系统（用户购物车 + 游客购物车）
- 订单管理与自动发货
- 钱包系统与余额充值
- 礼品卡兑换功能
- 推广联盟系统
- 上游 API 对接（商品同步、订单回调）
- 内容管理系统（文章、Banner）
- 系统配置管理
- RBAC 权限管理系统
- Docker 容器化部署支持
- 数据统计看板

### Security
- JWT 双令牌体系（管理员/用户独立密钥）
- AES-256-GCM 敏感数据加密
- 登录限流与 IP 黑名单
- 图片验证码 + 邮箱验证码双重验证
- 可配置的密码策略
- CORS 白名单控制

## [1.0.0] - 2026-02-10

### Added
- 初始版本发布
- 核心商品管理功能
- 基础订单管理
- 卡密自动发货
- 支付渠道集成
- 用户系统
- 管理后台

[Unreleased]: https://github.com/zhifengjin050-arch/dujiao-next/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/zhifengjin050-arch/dujiao-next/releases/tag/v1.0.0