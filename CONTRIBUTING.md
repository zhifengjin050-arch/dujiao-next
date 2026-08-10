# 贡献指南

感谢你对 Dujiao-Next 的关注！我们欢迎任何形式的贡献，包括但不限于：

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码修复
- ✨ 实现新功能

## 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [Issue 规范](#issue-规范)

## 行为准则

本项目遵循 [Contributor Covenant 行为准则](CODE_OF_CONDUCT.md)。参与即表示您同意遵守其条款。

## 如何贡献

### 报告 Bug

1. 先在 [Issues](https://github.com/zhifengjin050-arch/dujiao-next/issues) 中搜索，确认是否已有相同问题
2. 使用 Bug Report 模板创建 Issue
3. 提供详细的复现步骤、期望行为和实际行为
4. 附上相关的环境信息（操作系统、Go 版本、Node.js 版本等）

### 提出新功能

1. 先在 Issues 中搜索，确认是否已有类似建议
2. 使用 Feature Request 模板创建 Issue
3. 清晰描述功能的使用场景和价值
4. 如果可能，提供技术方案的初步设想

## 开发环境搭建

### 环境要求

| 依赖 | 最低版本 | 推荐版本 |
|------|---------|---------|
| Go | 1.24 | 1.25+ |
| Node.js | 18 | 20+ |
| Redis | 6.0 | 7.0+ |
| Git | 2.30 | 最新版 |

### 克隆并启动

```bash
# 克隆仓库
git clone https://github.com/zhifengjin050-arch/dujiao-next.git
cd dujiao-next

# 启动后端 API
cd api
cp config.yml.example config.yml
go mod tidy
go run cmd/server/main.go

# 启动管理后台（新终端）
cd admin
npm install
npm run dev

# 启动用户前台（新终端）
cd user
npm install
npm run dev
```

## 代码规范

### Go 代码规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行静态检查
- 所有导出函数和类型必须有注释
- 错误处理必须明确，不允许忽略错误
- 使用有意义的变量名，避免单字母变量（循环变量除外）

```bash
# 格式化
gofmt -w ./...

# 静态检查
golangci-lint run ./...
```

### Vue/TypeScript 代码规范

- 遵循 [Vue Style Guide](https://vuejs.org/style-guide/) 优先级 A 和 B 规则
- 使用 ESLint + Prettier 进行代码格式化
- 组件使用 `<script setup lang="ts">` 语法
- 使用 Composition API，避免 Options API
- 复杂逻辑抽取为 Composables

```bash
# 格式化
npm run lint

# 类型检查
npm run type-check
```

### 通用规范

- 所有公开接口必须有注释
- 复杂业务逻辑需要添加说明注释
- 敏感信息（密钥、密码等）不得硬编码
- 提交前确保所有测试通过
- 新增功能需要添加对应的测试用例

## 提交规范

本项目使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

### 提交消息格式

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### 类型 (type)

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响代码运行） |
| `refactor` | 重构（既不是新功能也不是 Bug 修复） |
| `perf` | 性能优化 |
| `test` | 添加或修改测试 |
| `chore` | 构建过程或辅助工具的变动 |
| `ci` | CI/CD 相关变更 |
| `build` | 影响构建系统或外部依赖的变更 |

### 范围 (scope)

| 范围 | 说明 |
|------|------|
| `api` | 后端 API 服务 |
| `admin` | 管理后台前端 |
| `user` | 用户前台前端 |
| `payment` | 支付模块 |
| `auth` | 认证模块 |
| `order` | 订单模块 |
| `product` | 商品模块 |
| `config` | 配置相关 |
| `docker` | Docker 部署相关 |
| `docs` | 文档相关 |

### 示例

```bash
git commit -m "feat(payment): add Stripe payment provider"
git commit -m "fix(order): correct order status transition on refund"
git commit -m "docs(api): update API documentation for order endpoints"
git commit -m "refactor(admin): extract product form into reusable component"
```

## Pull Request 流程

1. **Fork 仓库** 并克隆到本地

2. **创建分支**
   ```bash
   git checkout -b feat/your-feature-name
   # 或
   git checkout -b fix/your-bug-fix
   ```

3. **开发和测试**
   - 编写代码
   - 确保代码符合规范
   - 添加必要的测试
   - 确保所有测试通过

4. **提交代码**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   git push origin feat/your-feature-name
   ```

5. **创建 Pull Request**
   - 前往 GitHub 仓库页面创建 PR
   - 填写 PR 模板，清晰描述变更内容
   - 关联相关的 Issue

6. **代码审查**
   - 维护者会审查你的代码
   - 根据反馈进行修改
   - 审查通过后合并

### PR 检查清单

- [ ] 代码符合项目规范
- [ ] 所有测试通过
- [ ] 新功能有对应的测试
- [ ] 文档已更新
- [ ] 提交消息符合 Conventional Commits 规范
- [ ] 没有引入新的安全漏洞
- [ ] 敏感信息已妥善处理

## Issue 规范

### Bug Report 模板

```markdown
### 描述
清晰描述 Bug 的表现

### 复现步骤
1. 打开 '...'
2. 点击 '...'
3. 看到错误 '...'

### 期望行为
描述你期望发生什么

### 截图
如果适用，添加截图

### 环境信息
- 操作系统: [e.g. Ubuntu 22.04]
- Go 版本: [e.g. 1.25.0]
- Node.js 版本: [e.g. 20.11.0]
- 浏览器: [e.g. Chrome 120]
```

### Feature Request 模板

```markdown
### 问题描述
清晰描述这个功能解决什么问题

### 解决方案
描述你期望的解决方案

### 替代方案
描述你考虑过的替代方案

### 附加信息
添加任何其他上下文或截图
```

## 社区

- 📧 邮箱: [项目维护者邮箱]
- 💬 讨论区: [GitHub Discussions](https://github.com/zhifengjin050-arch/dujiao-next/discussions)

---

感谢你的贡献！🦄