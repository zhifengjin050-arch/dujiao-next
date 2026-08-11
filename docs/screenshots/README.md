# 项目截图

本目录用于存放项目关键界面的截图。请将实际截图文件放入此目录，并在下方表格中更新对应的文件路径。

## 截图清单

| 序号 | 界面 | 文件名 | 说明 |
|------|------|--------|------|
| 1 | 管理后台 - 数据看板 | `01-admin-dashboard.png` | 销售统计、订单趋势、支付渠道分析 |
| 2 | 管理后台 - 商品管理 | `02-admin-products.png` | 多规格商品（SKU）管理、库存管理 |
| 3 | 管理后台 - 订单管理 | `03-admin-orders.png` | 订单列表、详情、手动发货、导出 |
| 4 | 用户前台 - 商品浏览 | `04-user-products.png` | 商品列表、分类筛选、搜索 |
| 5 | 用户前台 - 购物车与结算 | `05-user-checkout.png` | 购物车管理、多支付渠道选择 |

## 如何截图

### 启动开发环境

```bash
# 1. 启动后端 API
cd api
cp config.yml.example config.yml
go run cmd/server/main.go

# 2. 启动管理后台（新终端）
cd admin
npm install && npm run dev

# 3. 启动用户前台（新终端）
cd user
npm install && npm run dev
```

### 访问地址

| 端 | 地址 |
|----|------|
| 管理后台 | http://localhost:5174 |
| 用户前台 | http://localhost:5173 |

### 截图要求

- 分辨率：1920x1080 或更高
- 格式：PNG
- 内容：展示系统核心功能，确保界面完整、数据真实
- 隐私：避免截图中包含真实用户数据或敏感信息
