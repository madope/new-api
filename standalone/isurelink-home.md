# ISURELINK 独立首页

## 文件

- HTML: `standalone/isurelink-home.html`

## 用法

本页是单文件落地页，不依赖额外 CSS、字体或 JS 资源。右侧登录区会通过 iframe 加载 `new-api` 前端新增的 `/embedded-sign-in` 页面。

### 1. 本地预览

直接用浏览器打开 `standalone/isurelink-home.html` 即可预览静态外观。

如果要让右侧登录区正常加载，请带上 `base` 查询参数，例如：

```text
file:///.../standalone/isurelink-home.html?base=https://your-domain.com
```

或部署到 HTTP 服务后访问：

```text
https://your-static-host/isurelink-home.html?base=https://your-domain.com
```

### 2. 参数

- `base`: 目标 `new-api` 站点地址，页面会自动拼接 `/embedded-sign-in`
- `path`: 可选，自定义登录路径，默认 `/embedded-sign-in`
- `redirect`: 可选，透传到登录页查询参数
- `theme`: 可选，`light` 或 `dark`
- `lang`: 可选，设置页面根节点语言标记

示例：

```text
https://your-static-host/isurelink-home.html?base=https://api.example.com&theme=dark
```

### 3. 配置到系统设置

将可访问的页面 URL 填入系统设置中的 `Home Page Content`。

推荐填写完整 URL，例如：

```text
https://your-static-host/isurelink-home.html?base=https://api.example.com
```

当前首页逻辑会把这个 URL 当作 iframe 页面嵌入显示。

## 注意事项

- 本方案需要前端提供 `/embedded-sign-in` 路由。
- 右侧 iframe 默认使用 `/embedded-sign-in`，该路由复用现有登录页的核心组件与逻辑，但去掉整页壳层。
- 如果目标站点设置了禁止 iframe 加载的安全头，右侧登录区将无法显示。
