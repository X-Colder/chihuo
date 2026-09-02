# 吃货原生骑手小程序

这是第一阶段的微信原生骑手端，使用 WXML、WXSS 和 JavaScript 编写，可直接导入微信开发者工具。

## 导入调试

1. 打开微信开发者工具，选择“导入项目”。
2. 选择当前目录：`miniprograms/rider`。
3. 没有正式 AppID 时可暂时使用 `project.config.json` 中的 `touristappid`；发布前替换为正式小程序 AppID。
4. 在“详情”中勾选“不校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书”，方便本地联调。
5. 点击“编译”，默认进入“配送任务”页面。

## API 地址配置

默认 API 地址写在 `app.js` 的 `API_BASE_URL`：

```js
globalData: {
  API_BASE_URL: 'http://localhost:4000'
}
```

本地开发者工具通常可以访问 `localhost`。真机调试不能把手机上的 `localhost` 当作电脑 API 地址，需要改为局域网 IP 或已备案的 HTTPS 域名，并配置微信公众平台的 request 合法域名。

也可以在开发者工具控制台设置运行时地址：

```js
wx.setStorageSync('chihuo.rider.apiBaseUrl', 'http://127.0.0.1:4000')
```

清除运行时地址：

```js
wx.removeStorageSync('chihuo.rider.apiBaseUrl')
```

## 认证与接口

`utils/request.js` 使用 `wx.request`，本地调试时第一次请求会调用开发登录接口：

```text
POST /v1/auth/dev/wechat-login
```

生产环境将 `app.js` 中的 `LOGIN_MODE` 改为 `wechat`，请求
`POST /v1/auth/wechat-login`。AppSecret 只配置在 Go API 服务端。

返回的 JWT 会保存到：

```text
chihuo.rider.token
```

后续请求自动添加 `Authorization: Bearer <token>`。

当前页面使用的 API：

| 用途 | 方法 | 路径 |
| --- | --- | --- |
| 已分配任务 | GET | `/v1/rider/tasks` |
| 可领取任务 | GET | `/v1/rider/tasks/queue` |
| 领取任务 | POST | `/v1/rider/tasks/:id/claim` |
| 更新任务状态 | PATCH | `/v1/rider/tasks/:id` |

## 页面能力

- 任务列表、状态筛选、下拉刷新；
- 领取任务；
- 确认取餐、开始配送、确认送达；
- 任务详情、地址、餐品和备注展示；
- 配送异常上报占位；
- API 不可用时使用本地演示数据；
- Token 和 API 地址使用微信 storage 保存。

异常上报目前复用 `PATCH /v1/rider/tasks/:id`，将任务标记为 `CANCELLED` 并写入备注。这是第一阶段占位方案，后续接入独立的骑手异常事件接口后，只需替换详情页提交逻辑。
