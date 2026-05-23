# DanQing UI in Teams

与 [DanQing-Studio](../DanQing-Studio/frontend/DQ-UI.md) 一致，使用共享包 [`../dq-ui`](../dq-ui)。

## 栈

| 层 | 包 |
|----|-----|
| Tokens | `@danqing/dq-tokens` |
| 组件 | `@danqing/dq-ui`（`Dq*`） |
| Shell | `@danqing/dq-shell`（反馈、图标、部分壳组件） |

**禁止 Element Plus**：模板仅使用 `Dq*` 组件。

## 本地开发

```bash
cd ../dq-ui && pnpm install
cd ../DanQing-Teams/frontend && npm install && npm run dev
```

修改 `dq-ui` 后需重启 Vite。
