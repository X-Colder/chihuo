# 微信原生小程序调试

## 项目目录

```text
miniprograms/
├── consumer/
├── merchant/
└── rider/
```

每个目录都是独立的微信开发者工具项目，不使用 React、Vite 或浏览器 `index.html`。

## 本地调试

1. 打开微信开发者工具。
2. 分别导入 `miniprograms/consumer`、`miniprograms/merchant` 或 `miniprograms/rider`。
3. 在 `project.config.json` 中填写 AppID；本地调试可使用测试配置。
4. 在 `app.js` 设置 Go API 地址，例如 `http://127.0.0.1:4000`。
5. 真机调试时将 API 替换为 HTTPS 域名，并在微信公众平台配置 request 合法域名。

开发环境使用 Go 后端的 dev login；生产环境切换为微信 `wx.login` 码换取会话，不在小程序端保存 AppSecret。
