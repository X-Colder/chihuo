# 吃货消费者小程序

这是微信原生小程序项目，使用 WXML、WXSS 和 JavaScript 编写，不依赖 React、Vite 或 npm。

## 导入调试

在微信开发者工具中选择“导入项目”，项目目录选择当前目录：

```text
miniprograms/consumer
```

`project.config.json` 中的 `appid` 默认为空。使用正式 AppID 时，在微信开发者工具的项目设置中填写，或直接修改该文件的 `appid` 字段。

本地调试 Go API 时，请先启动后端，然后修改 `app.js`：

```js
globalData: {
  config: {
    API_BASE_URL: 'http://127.0.0.1:4000',
    DEV_LOGIN_PATH: '/v1/auth/dev/wechat-login'
  }
}
```

开发登录会先调用 `wx.login()` 获取 code，再调用 `DEV_LOGIN_PATH`。默认角色为 `CONSUMER`，登录返回的 token 和用户信息会保存到微信 storage。后续请求自动携带：

```text
Authorization: Bearer <token>
```

微信开发者工具本地联调时，可以在“详情 -> 本地设置”中关闭“校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书”。生产环境必须使用备案域名和 HTTPS，并在小程序后台配置 request 合法域名。

## 页面

- 需求广场：查看附近正在聚合的需求
- 发布需求：提交结构化需求和硬约束
- 需求详情：查看进度并加入需求
- 预售商品：浏览已确认规格的餐品并下单
- 订单：查看支付、制作和配送状态
