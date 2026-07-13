# Cute Jone 小程序 MVP

这是当前项目的小程序前端，复用现有后端和数据库，不会迁移或修改服务端数据结构。

## 维护约定

后续产品功能只维护微信小程序端，Web 端不再作为业务功能维护入口。新增或调整管理员能力时，也优先在小程序内实现管理员入口，例如公告发布、商品审核、用户管理等。

## 已包含页面

- 首页商品流：分类、搜索、排序、下拉刷新、触底加载
- 首页公告展示：展示当前启用公告，用户点关闭后仅本次小程序运行期间隐藏
- 失物招领：寻物 / 招领列表、发布、详情、私信联系和关闭信息
- 商品详情：图片、价格、卖家信息、收藏
- 发布闲置：分类、价格、成色、图片上传
- 登录 / 注册：沿用学号密码体系
- 我的：个人信息、我的收藏、我的发布、退出登录
- 小程序管理员公告管理：发布、编辑、启用、停用和删除首页公告
- 收藏列表
- 我的发布列表

## 打开方式

1. 安装微信开发者工具。
2. 复制本地项目配置：`cp project.config.example.json project.config.json`。
3. 在本地 `project.config.json` 中填写自己的微信小程序 App ID。该文件已被 Git 忽略，请勿提交真实 App ID 或 AppSecret。
4. 选择“导入项目”，项目目录选择本目录：`miniprogram/`。
5. 本地调试已关闭域名校验；如果商品列表为空，仍可在开发者工具“详情 -> 本地设置”确认“不校验合法域名”已勾选。
6. 真机预览或正式发布前，在小程序后台配置服务器域名：
   - request 合法域名：`https://dealinfor.drebel.top`
   - uploadFile 合法域名：`https://dealinfor.drebel.top`
   - downloadFile 合法域名：`https://dealinfor.drebel.top`

## 接口说明

接口根地址在 `utils/api.js`：

```js
const BASE_URL = 'https://dealinfor.drebel.top'
```

本小程序复用现有接口：

- `GET /api/products`
- `GET /api/products/{id}`
- `POST /api/products`
- `GET /api/categories`
- `POST /api/auth/login`
- `POST /api/auth/register`
- `GET /api/users/me`
- `POST /api/upload/image`
- `GET /api/favorites`
- `POST /api/favorites/{productId}`
- `DELETE /api/favorites/{productId}`

## 当前 MVP 限制

- 暂未接入微信 openid 登录，仍使用“学号 + 密码”，避免改动用户表。
- 已接入小程序内私信页面，商品详情里的“联系卖家”会进入和卖家的对话。
- 暂未做订单流程入口，先保证浏览、注册登录、发布和收藏可用。
- 公告发布入口已放在小程序“我的 -> 公告管理”，仅管理员账号可见。
