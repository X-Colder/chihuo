# 吃货商家小程序

这是微信原生小程序项目，使用 WXML、WXSS 和 JavaScript 编写，不依赖 React、Vite 或 npm。

## 导入调试

在微信开发者工具中选择“导入项目”，项目目录选择当前目录：

```text
miniprograms/merchant
```

`project.config.json` 中的 `appid` 默认为空。使用正式 AppID 时，在微信开发者工具的项目设置中填写，或直接修改该文件的 `appid` 字段。

本地调试 Go API 时，请先启动后端，然后修改 `app.js`：

```js
globalData: {
  config: {
    API_BASE_URL: 'http://127.0.0.1:4000',
    LOGIN_MODE: 'dev',
    LOGIN_PATH: '/v1/auth/wechat-login',
    DEV_LOGIN_PATH: '/v1/auth/dev/wechat-login',
    DEV_ROLE: 'MERCHANT',
    DEV_NAME: '开发商家',
    DEV_MERCHANT_NAME: '吃货开发演示餐厅'
  }
}
```

本地调试使用 `LOGIN_MODE: 'dev'`，会调用 `DEV_LOGIN_PATH`。生产环境改为
`LOGIN_MODE: 'wechat'`，调用 `LOGIN_PATH`。两种模式都会先调用
`wx.login()` 获取 code，登录返回的 token 和用户信息会保存到微信 storage。后续请求自动携带：

```text
Authorization: Bearer <token>
```

微信开发者工具本地联调时，可以在“详情 -> 本地设置”中关闭“校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书”。生产环境必须使用备案域名和 HTTPS，并在小程序后台配置 request 合法域名。

## 页面

- 需求大厅：查看附近的匿名化聚合需求
- 需求评估：查看生产参考和报价入口
- 提交报价：提交价格、产能、食材、过敏原和保存要求
- 预售活动：查看入选并审核后的活动
- 资质中心：资质状态和资料审核占位
- 食品安全：查看记录并提交风险报备
