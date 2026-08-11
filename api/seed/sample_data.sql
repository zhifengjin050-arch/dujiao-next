-- ============================================================================
-- dujiao-next 虚拟商品自动发货系统 - 示例/种子数据
-- 用途：开发环境初始化数据库，提供可用的示例分类、商品和 SKU 数据
-- 兼容：SQLite 语法
-- 使用方法：sqlite3 dujiao.db < sample_data.sql
-- 注意：该脚本假设表结构已通过 GORM AutoMigrate 创建，仅插入数据
-- ============================================================================

-- ============================================================================
-- 1. 分类数据 (categories)
-- ============================================================================

-- 一级分类：虚拟卡密
-- 适用于各类会员卡密、礼品卡、充值卡等虚拟卡密商品
INSERT INTO categories (id, parent_id, slug, name_json, icon, sort_order, created_at, deleted_at)
VALUES (1, 0, 'virtual-card-key', '{"zh": "虚拟卡密", "en": "Virtual Card Keys"}', '', 100, '2026-01-01 00:00:00', NULL);

-- 一级分类：游戏充值
-- 适用于各类游戏点券、游戏币、游戏道具等充值商品
INSERT INTO categories (id, parent_id, slug, name_json, icon, sort_order, created_at, deleted_at)
VALUES (2, 0, 'game-recharge', '{"zh": "游戏充值", "en": "Game Recharge"}', '', 90, '2026-01-01 00:00:00', NULL);

-- 一级分类：软件激活码
-- 适用于操作系统、办公软件、开发工具等正版激活码/许可证
INSERT INTO categories (id, parent_id, slug, name_json, icon, sort_order, created_at, deleted_at)
VALUES (3, 0, 'software-activation', '{"zh": "软件激活码", "en": "Software Activation"}', '', 80, '2026-01-01 00:00:00', NULL);

-- 一级分类：在线服务
-- 适用于远程运维、技术支持、咨询服务等人工交付类服务
INSERT INTO categories (id, parent_id, slug, name_json, icon, sort_order, created_at, deleted_at)
VALUES (4, 0, 'online-service', '{"zh": "在线服务", "en": "Online Services"}', '', 70, '2026-01-01 00:00:00', NULL);


-- ============================================================================
-- 2. 商品数据 (products)
-- ============================================================================

-- 商品1：腾讯视频VIP会员（虚拟卡密类，自动发货，多SKU）
-- 有多个 SKU（月卡/季卡/年卡），自动发货，价格区间 19.90 ~ 198.00
INSERT INTO products (id, category_id, slug, seo_meta_json, title_json, description_json, content_json, instructions_json, price_amount, cost_price_amount, images, tags, purchase_type, max_purchase_quantity, fulfillment_type, manual_form_schema_json, manual_stock_total, manual_stock_locked, manual_stock_sold, payment_channel_ids, is_affiliate_enabled, is_mapped, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (1, 1, 'sample-tencent-video-vip',
    '{"zh": {"title": "腾讯视频VIP会员-示例商品", "description": "腾讯视频VIP会员自动发货，多规格可选"}}',
    '{"zh": "示例商品-腾讯视频VIP会员", "en": "Sample - Tencent Video VIP"}',
    '{"zh": "腾讯视频VIP会员，自动发货，秒到账。支持月卡、季卡、年卡多种规格。", "en": "Tencent Video VIP membership, auto delivery, instant activation."}',
    '{"zh": "## 商品详情\n\n- 官方正品卡密，自动发货\n- 支持手机/电脑/平板多端观看\n- 海量影视资源，高清无广告"}',
    '{"zh": "## 使用说明\n\n1. 打开腾讯视频APP\n2. 进入「我的」-「VIP会员」\n3. 点击「兑换会员」\n4. 输入收到的卡密兑换"}',
    '19.90', '15.00', '["https://example.com/images/tencent-video.jpg"]', '["视频会员","VIP","自动发货"]',
    'member', 5, 'auto', '{}', -1, 0, 0, '', 1, 0, 1, 100, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- 商品2：王者荣耀点券充值（游戏充值类，自动发货，多SKU）
-- 有多个面值 SKU（50元/100元/200元/500元），自动发货
INSERT INTO products (id, category_id, slug, seo_meta_json, title_json, description_json, content_json, instructions_json, price_amount, cost_price_amount, images, tags, purchase_type, max_purchase_quantity, fulfillment_type, manual_form_schema_json, manual_stock_total, manual_stock_locked, manual_stock_sold, payment_channel_ids, is_affiliate_enabled, is_mapped, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (2, 2, 'sample-wzry-points',
    '{"zh": {"title": "王者荣耀点券充值-示例商品", "description": "王者荣耀点券自动充值，多面值可选"}}',
    '{"zh": "示例商品-王者荣耀点券充值", "en": "Sample - Honor of Kings Points"}',
    '{"zh": "王者荣耀点券自动充值，多面值规格可选，秒到账。", "en": "Honor of Kings points auto recharge, multiple denominations."}',
    '{"zh": "## 商品详情\n\n- 官方渠道充值，安全可靠\n- 自动发货，秒到账\n- 支持QQ区/微信区"}',
    '{"zh": "## 使用说明\n\n1. 下单时填写正确的游戏账号\n2. 选择对应区服\n3. 支付后点券自动到账"}',
    '50.00', '45.00', '["https://example.com/images/wzry.jpg"]', '["游戏充值","点券","王者荣耀","自动发货"]',
    'member', 10, 'auto', '{}', -1, 0, 0, '', 1, 0, 1, 95, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- 商品3：Office 365 激活码（软件激活码类，自动发货，单SKU）
-- 仅一个 SKU，自动发货，固定价格
INSERT INTO products (id, category_id, slug, seo_meta_json, title_json, description_json, content_json, instructions_json, price_amount, cost_price_amount, images, tags, purchase_type, max_purchase_quantity, fulfillment_type, manual_form_schema_json, manual_stock_total, manual_stock_locked, manual_stock_sold, payment_channel_ids, is_affiliate_enabled, is_mapped, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (3, 3, 'sample-office365-key',
    '{"zh": {"title": "Office 365激活码-示例商品", "description": "正版Office 365激活码，支持多设备"}}',
    '{"zh": "示例商品-Office 365 激活码", "en": "Sample - Office 365 License Key"}',
    '{"zh": "正版Office 365激活码，支持1台PC+1台平板+1台手机，含1TB OneDrive云存储。", "en": "Genuine Office 365 license key, supports multiple devices."}',
    '{"zh": "## 商品详情\n\n- 正版授权，永久激活\n- 支持1台PC + 1台平板 + 1台手机\n- 含1TB OneDrive云存储\n- 包含Word、Excel、PowerPoint等全套办公软件"}',
    '{"zh": "## 激活步骤\n\n1. 访问 office.com/setup\n2. 登录你的Microsoft账号\n3. 输入激活码完成激活\n4. 下载安装Office套件"}',
    '299.00', '200.00', '["https://example.com/images/office365.jpg"]', '["软件","激活码","Office","正版"]',
    'member', 3, 'auto', '{}', 100, 0, 0, '', 0, 0, 1, 85, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- 商品4：服务器运维服务（在线服务类，人工交付，单SKU）
-- 人工交付，需要填写表单，有限库存
INSERT INTO products (id, category_id, slug, seo_meta_json, title_json, description_json, content_json, instructions_json, price_amount, cost_price_amount, images, tags, purchase_type, max_purchase_quantity, fulfillment_type, manual_form_schema_json, manual_stock_total, manual_stock_locked, manual_stock_sold, payment_channel_ids, is_affiliate_enabled, is_mapped, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (4, 4, 'sample-server-maintenance',
    '{"zh": {"title": "服务器运维服务-示例商品", "description": "专业服务器运维，安全加固，性能优化"}}',
    '{"zh": "示例商品-服务器运维服务", "en": "Sample - Server Maintenance Service"}',
    '{"zh": "专业服务器运维服务，包含安全加固、性能优化、日常监控等。", "en": "Professional server maintenance service including security hardening and performance optimization."}',
    '{"zh": "## 服务内容\n\n- 服务器安全加固\n- 性能优化调优\n- 日常监控告警配置\n- 7x24小时技术支持"}',
    '{"zh": "## 服务流程\n\n1. 下单后提供服务器信息\n2. 工程师评估并制定方案\n3. 执行运维操作\n4. 交付运维报告"}',
    '500.00', '300.00', '["https://example.com/images/server.jpg"]', '["运维","服务器","技术服务"]',
    'member', 1, 'manual', '{"fields": [{"name": "server_ip", "label": "服务器IP", "type": "text", "required": true}, {"name": "server_type", "label": "服务器类型", "type": "select", "options": ["Linux", "Windows"], "required": true}]}', 50, 0, 0, '', 0, 0, 1, 80, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- 商品5：Steam游戏充值卡（游戏充值类，自动发货，多SKU）
-- 多个面值 SKU，自动发货，无限库存
INSERT INTO products (id, category_id, slug, seo_meta_json, title_json, description_json, content_json, instructions_json, price_amount, cost_price_amount, images, tags, purchase_type, max_purchase_quantity, fulfillment_type, manual_form_schema_json, manual_stock_total, manual_stock_locked, manual_stock_sold, payment_channel_ids, is_affiliate_enabled, is_mapped, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (5, 2, 'sample-steam-gift-card',
    '{"zh": {"title": "Steam充值卡-示例商品", "description": "Steam钱包充值卡，多面值，自动发货"}}',
    '{"zh": "示例商品-Steam游戏充值卡", "en": "Sample - Steam Gift Card"}',
    '{"zh": "Steam钱包充值卡，多面值可选，自动发货秒到账，支持全球区服。", "en": "Steam wallet gift card, multiple denominations, instant delivery."}',
    '{"zh": "## 商品详情\n\n- 官方正品充值卡\n- 自动发货，秒到账\n- 支持全球区服\n- 可用于购买游戏、DLC、道具等"}',
    '{"zh": "## 使用说明\n\n1. 打开Steam客户端\n2. 点击右上角用户名 → 「账户明细」\n3. 点击「为您的Steam钱包充值」\n4. 选择「兑换Steam充值卡」\n5. 输入收到的充值码"}',
    '50.00', '45.00', '["https://example.com/images/steam.jpg"]', '["游戏","Steam","充值卡","自动发货"]',
    'guest', 0, 'auto', '{}', -1, 0, 0, '', 1, 0, 1, 90, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);


-- ============================================================================
-- 3. SKU 数据 (product_skus)
-- 说明：price_amount 为售价，cost_price_amount 为成本价，manual_stock_total 为库存（-1 表示无限库存）
--       is_active 为 1 表示启用，sort_order 控制排序，越大越靠前
-- ============================================================================

-- ---------------------------------------------------------------------------
-- 商品1（腾讯视频VIP会员）的 SKU：月卡、季卡、年卡
-- ---------------------------------------------------------------------------
INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (1, 1, 'VIP-MONTH', '{"zh": "月卡", "en": "Monthly"}', '19.90', '15.00', -1, 0, 0, 1, 10, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (2, 1, 'VIP-QUARTER', '{"zh": "季卡", "en": "Quarterly"}', '55.00', '42.00', -1, 0, 0, 1, 9, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (3, 1, 'VIP-YEAR', '{"zh": "年卡", "en": "Yearly"}', '198.00', '160.00', -1, 0, 0, 1, 8, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- ---------------------------------------------------------------------------
-- 商品2（王者荣耀点券）的 SKU：50元、100元、200元、500元面值
-- ---------------------------------------------------------------------------
INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (4, 2, 'WZRY-50', '{"zh": "面值50元", "en": "50 CNY"}', '50.00', '45.00', -1, 0, 0, 1, 10, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (5, 2, 'WZRY-100', '{"zh": "面值100元", "en": "100 CNY"}', '100.00', '90.00', -1, 0, 0, 1, 9, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (6, 2, 'WZRY-200', '{"zh": "面值200元", "en": "200 CNY"}', '200.00', '180.00', -1, 0, 0, 1, 8, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (7, 2, 'WZRY-500', '{"zh": "面值500元", "en": "500 CNY"}', '500.00', '450.00', -1, 0, 0, 1, 7, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- ---------------------------------------------------------------------------
-- 商品3（Office 365 激活码）的 SKU：仅一个默认规格（个人版一年）
-- ---------------------------------------------------------------------------
INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (8, 3, 'DEFAULT', '{"zh": "个人版一年", "en": "Personal 1-Year"}', '299.00', '200.00', 100, 0, 0, 1, 10, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- ---------------------------------------------------------------------------
-- 商品4（服务器运维服务）的 SKU：根据服务级别分为基础版和高级版
-- 人工交付，有限库存
-- ---------------------------------------------------------------------------
INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (9, 4, 'SVR-BASIC', '{"zh": "基础运维服务", "en": "Basic Maintenance"}', '500.00', '300.00', 30, 0, 0, 1, 10, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (10, 4, 'SVR-PRO', '{"zh": "高级运维服务", "en": "Pro Maintenance"}', '1200.00', '800.00', 20, 0, 0, 1, 9, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

-- ---------------------------------------------------------------------------
-- 商品5（Steam充值卡）的 SKU：50元、100元、300元面值
-- 自动发货，无限库存，支持游客购买
-- ---------------------------------------------------------------------------
INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (11, 5, 'STEAM-50', '{"zh": "面值50元", "en": "50 CNY"}', '50.00', '45.00', -1, 0, 0, 1, 10, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (12, 5, 'STEAM-100', '{"zh": "面值100元", "en": "100 CNY"}', '100.00', '90.00', -1, 0, 0, 1, 9, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);

INSERT INTO product_skus (id, product_id, sku_code, spec_values_json, price_amount, cost_price_amount, manual_stock_total, manual_stock_locked, manual_stock_sold, is_active, sort_order, created_at, updated_at, deleted_at)
VALUES (13, 5, 'STEAM-300', '{"zh": "面值300元", "en": "300 CNY"}', '300.00', '270.00', -1, 0, 0, 1, 8, '2026-01-01 00:00:00', '2026-01-01 00:00:00', NULL);
