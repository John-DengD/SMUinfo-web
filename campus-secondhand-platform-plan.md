# 校园二手交易平台建设方案

## 1. 项目定位

本项目是面向校内学生的二手交易平台，核心目标是为学生提供一个发布、浏览、搜索、收藏、沟通和线下交易二手物品的校园交易场景。

平台可以参考闲鱼类二手交易网站的页面结构和交互方式，但业务重点应放在校园内的可信交易、学生身份注册、线下见面交易和轻量化管理上。

本平台不接入 Stripe、支付宝、微信支付等真实线上支付能力，所有付款都在线下完成。平台只负责信息撮合、沟通记录、交易状态管理和后台审核。

## 2. 核心用户

- 普通学生：注册、登录、浏览商品、搜索商品、收藏商品、联系卖家、发布商品、管理自己的交易。
- 卖家学生：发布闲置物品、管理商品状态、回复买家消息、确认交易。
- 平台管理员：管理用户、分类、商品、举报信息和违规内容。

## 3. 技术架构

### 3.1 推荐技术栈

前端：

```text
Vue 3 + Vite + Vue Router + Pinia + Axios + Element Plus
```

后端：

```text
Spring Boot 3 + Spring Security + JWT + MyBatis Plus
```

数据库：

```text
MySQL 8
```

图片存储：

```text
开发阶段：本地文件上传目录
后期扩展：阿里云 OSS / 腾讯云 COS / MinIO
```

部署：

```text
Nginx + Spring Boot Jar + MySQL
```

### 3.2 架构分层

```text
前端页面
  |
  | HTTP API / JWT
  |
后端服务
  |
  | MyBatis Plus
  |
MySQL 数据库
```

后端负责用户认证、商品管理、交易管理、消息管理、权限控制和数据持久化。

前端负责页面展示、用户交互、表单提交、登录状态维护和接口调用。

## 4. 功能模块设计

### 4.1 用户注册与登录

注册字段：

- 姓名
- 学号
- 密码
- 手机号，可选
- 学院，可选
- 校区，可选

规则：

- 学号必须唯一。
- 密码不能明文存储，后端需要加密保存。
- 登录成功后由后端签发 JWT Token。
- 前端保存 Token，并在后续接口请求中携带。

### 4.2 首页

首页是平台的主要流量入口，建议包括：

- 顶部导航栏
- Logo
- 搜索框
- 发布按钮
- 消息入口
- 我的入口
- 分类导航
- 校园活动 Banner
- 推荐商品流

商品卡片展示：

- 商品图片
- 商品标题
- 当前价格
- 商品成色
- 校区或交易地点
- 发布时间

### 4.3 商品分类

初始分类建议：

- 教材资料
- 电子数码
- 宿舍用品
- 服装鞋包
- 运动户外
- 美妆个护
- 交通工具
- 票券卡券
- 其他闲置

### 4.4 商品发布

学生登录后可以发布商品。

商品字段：

- 商品标题
- 商品描述
- 商品分类
- 当前价格
- 原价，可选
- 商品成色
- 交易地点
- 商品图片

发布后商品默认进入“在售”状态。管理员可以在后台下架违规商品。

### 4.5 商品详情

商品详情页展示：

- 商品图片轮播
- 商品标题
- 价格
- 原价
- 成色
- 商品描述
- 卖家信息
- 交易地点
- 发布时间
- 浏览次数
- 收藏按钮
- 联系卖家按钮
- 我想要按钮

### 4.6 搜索与筛选

支持按以下条件搜索：

- 关键词
- 商品分类
- 价格区间
- 商品成色
- 发布时间
- 校区或交易地点

搜索结果可以按发布时间、价格、热度排序。

### 4.7 收藏功能

用户可以收藏感兴趣的商品，并在“我的收藏”中查看。

如果商品已售出或下架，收藏列表中需要显示对应状态。

### 4.8 消息功能

第一版可以先做站内消息，不必一开始做实时聊天。

消息功能包括：

- 买家给卖家留言
- 卖家回复买家
- 按商品维度查看沟通记录
- 未读消息提醒

后期可以升级为 WebSocket 实时聊天。

### 4.9 线下交易流程

因为平台不接入真实支付，所以交易流程应设计为线下完成：

```text
买家点击“我想要”
  -> 系统创建交易请求
  -> 卖家确认预约
  -> 双方通过消息沟通时间和地点
  -> 校园内线下见面验货
  -> 买家线下付款
  -> 双方或卖家确认完成
  -> 商品状态变为已售出
```

交易状态建议：

- 待确认
- 已预约
- 已完成
- 已取消

商品状态建议：

- 在售
- 已预约
- 已售出
- 已下架

### 4.10 管理后台

后台功能包括：

- 用户管理
- 商品管理
- 分类管理
- 举报管理
- Banner 管理
- 数据概览

管理员可以执行：

- 禁用违规用户
- 下架违规商品
- 处理举报
- 新增或修改分类
- 管理首页 Banner

## 5. 数据库设计

### 5.1 用户表 user

```text
id
name
student_no
password_hash
phone
college
campus
avatar
role
status
created_at
updated_at
```

说明：

- student_no 需要加唯一索引。
- password_hash 存储加密后的密码。
- role 用于区分普通用户和管理员。
- status 用于标记正常、禁用等状态。

### 5.2 分类表 category

```text
id
name
icon
sort_order
status
created_at
updated_at
```

### 5.3 商品表 product

```text
id
seller_id
category_id
title
description
price
original_price
condition_level
trade_location
status
view_count
created_at
updated_at
```

说明：

- seller_id 关联用户表。
- category_id 关联分类表。
- status 标记在售、已预约、已售出、已下架。

### 5.4 商品图片表 product_image

```text
id
product_id
image_url
sort_order
created_at
```

一个商品可以有多张图片。

### 5.5 收藏表 favorite

```text
id
user_id
product_id
created_at
```

user_id 和 product_id 可以建立联合唯一索引，防止重复收藏。

### 5.6 消息表 message

```text
id
sender_id
receiver_id
product_id
content
is_read
created_at
```

### 5.7 交易表 trade_order

```text
id
product_id
buyer_id
seller_id
status
meet_location
remark
created_at
updated_at
completed_at
```

### 5.8 举报表 report

```text
id
reporter_id
product_id
reason
status
admin_remark
created_at
updated_at
```

### 5.9 评价表 review，可后期扩展

```text
id
order_id
reviewer_id
target_user_id
rating
content
created_at
```

## 6. 后端接口规划

### 6.1 用户接口

```text
POST /api/auth/register        用户注册
POST /api/auth/login           用户登录
GET  /api/users/me             获取当前用户信息
PUT  /api/users/me             修改当前用户信息
```

### 6.2 商品接口

```text
GET    /api/products           商品列表
GET    /api/products/{id}      商品详情
POST   /api/products           发布商品
PUT    /api/products/{id}      修改商品
DELETE /api/products/{id}      删除或下架商品
```

### 6.3 分类接口

```text
GET /api/categories            获取分类列表
```

### 6.4 图片上传接口

```text
POST /api/upload/image         上传商品图片
```

### 6.5 收藏接口

```text
POST   /api/favorites/{productId}    收藏商品
DELETE /api/favorites/{productId}    取消收藏
GET    /api/favorites                我的收藏
```

### 6.6 消息接口

```text
POST /api/messages             发送消息
GET  /api/messages             我的消息列表
GET  /api/messages/conversation/{userId}  获取会话记录
```

### 6.7 交易接口

```text
POST /api/orders               创建交易请求
GET  /api/orders               我的交易列表
PUT  /api/orders/{id}/confirm  确认预约
PUT  /api/orders/{id}/finish   确认完成
PUT  /api/orders/{id}/cancel   取消交易
```

### 6.8 管理后台接口

```text
GET /api/admin/users           用户管理
PUT /api/admin/users/{id}/status
GET /api/admin/products        商品管理
PUT /api/admin/products/{id}/status
GET /api/admin/reports         举报列表
PUT /api/admin/reports/{id}    处理举报
```

## 7. 前端页面规划

普通用户页面：

- 登录页
- 注册页
- 首页
- 分类页
- 搜索结果页
- 商品详情页
- 发布商品页
- 我的页面
- 我的发布
- 我的收藏
- 我的交易
- 消息中心

管理员页面：

- 后台首页
- 用户管理
- 商品管理
- 分类管理
- 举报管理
- Banner 管理

## 8. UI 风格建议

整体风格可以参考二手交易平台的高密度商品流设计，但需要做出校园特色。

建议风格：

- 主色使用清爽的绿色、蓝色或青色。
- 首页突出搜索和发布入口。
- 商品卡片信息清晰，价格醒目。
- 分类入口使用图标加文字。
- 页面布局简洁，避免过多营销化内容。
- 移动端优先，因为学生更可能使用手机访问。

首页结构建议：

```text
顶部导航
搜索区域
分类入口
校园 Banner
推荐商品流
```

## 9. 开发阶段规划

### 第一阶段：基础框架

目标：搭建项目基础能力。

任务：

- 创建前端 Vue 项目
- 创建后端 Spring Boot 项目
- 配置 MySQL
- 设计数据库表
- 完成前后端跨域配置
- 完成基础接口返回格式

### 第二阶段：用户系统

目标：完成注册、登录和身份认证。

任务：

- 用户注册
- 用户登录
- 密码加密
- JWT 认证
- 登录拦截
- 当前用户信息接口

### 第三阶段：商品系统

目标：完成商品发布和浏览闭环。

任务：

- 分类列表
- 发布商品
- 商品图片上传
- 商品列表
- 商品详情
- 商品状态管理

### 第四阶段：交易与互动

目标：完成买卖双方的沟通和线下交易流程。

任务：

- 收藏商品
- 发送消息
- 我的消息
- 创建交易请求
- 确认预约
- 确认完成
- 取消交易

### 第五阶段：管理后台

目标：让平台具备基本运营和审核能力。

任务：

- 用户管理
- 商品管理
- 举报处理
- 分类管理
- Banner 管理

### 第六阶段：优化与部署

目标：提升可用性并完成上线。

任务：

- 页面适配移动端
- 表单校验
- 错误提示
- 接口权限检查
- 数据库索引优化
- Nginx 部署
- 后端 Jar 部署
- MySQL 初始化脚本

## 10. MVP 最小可行版本

第一版建议只实现以下功能：

- 学生注册
- 学生登录
- 发布商品
- 上传商品图片
- 首页浏览商品
- 搜索商品
- 商品详情
- 收藏商品
- 联系卖家
- 创建线下交易请求
- 标记商品已售出
- 管理员下架违规商品

这个版本已经可以形成完整的校园二手交易业务闭环。

## 11. 后期扩展方向

后期可以继续扩展：

- 实时聊天
- 商品推荐
- 浏览历史
- 卖家信用分
- 交易评价
- 举报风控
- 校园认证
- 邮箱验证码
- 手机验证码
- 多校区切换
- 数据统计大屏
- 小程序端

## 12. 风险与注意事项

- 不要保存用户明文密码。
- 学号注册需要防止重复注册。
- 商品图片上传要限制文件大小和文件类型。
- 商品描述要防止 XSS 攻击。
- 管理后台接口必须做管理员权限校验。
- 线下交易需要明确平台不代收款、不担保交易。
- 已售出商品不能继续创建交易。
- 用户被禁用后不能登录、发布商品或发送消息。

## 13. 推荐实施方案

建议最终采用：

```text
前端：Vue 3 + Vite + Pinia + Vue Router + Axios + Element Plus
后端：Spring Boot 3 + Spring Security + JWT + MyBatis Plus
数据库：MySQL 8
图片：本地文件上传，后期可迁移到对象存储
部署：Nginx + Spring Boot Jar + MySQL
```

该方案适合课程设计、毕业设计和校内真实试点，功能完整、技术成熟、扩展空间明确。
