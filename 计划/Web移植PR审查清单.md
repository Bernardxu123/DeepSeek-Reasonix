# DeepSeek-Reasonix 桌面端 Web 移植 PR 审查清单

> 面向代码审查者的完整变更说明。关联文档：`Web移植技术总结.md` · `implementation_plan.md` · `task.md`
> 最后更新：2026-05-20（CI 已全部通过）

---

## 当前状态

| 项 | 状态 |
| --- | --- |
| CI / build (ubuntu + windows) | ✅ 通过 |
| CodeQL | ✅ 通过 |
| 分支冲突 | ✅ 已 rebase origin/main |
| 代码审查 | ⏳ 待开始 |

---

## 一、变更概览

| 类别 | 文件数 | 说明 |
| --- | --- | --- |
| 新增 React Dashboard | 17+ | `src/App.tsx`、`tauri-bridge.ts`、`ui/*`、`styles.css` 等 |
| 新增 CI 配置 | 1 | `.github/workflows/ci.yml` 增加 dashboard 依赖安装 + 构建顺序 |
| 新增测试 | 1 | `tests/dashboard-smoke.test.ts` |
| 删除旧 Preact | 38 | `dashboard/src/panels/*`、`dashboard/src/lib/*` |
| 删除过期测试 | 8 | 引用已删除模块的测试 |
| 修改核心文件 | 10+ | 后端路由、SSE 事件、设置 API、tsup 配置等 |
| 文档 | 5 | `计划/` 目录下设计文档 + 审查清单 |

---

## 二、架构设计（评审重点）

### 双模式桥接器 `tauri-bridge.ts`

这是整个迁移的**核心模块**。它使同一个 React App 在两种环境中运行：

| 模式 | 触发条件 | 数据来源 | 通信方式 |
| --- | --- | --- | --- |
| **Mock** | `<meta reasonix-mode>` = `__REASONIX_MODE__`（Vite 开发） | 内置模拟数据 | 直接 `broadcast()` |
| **Server** | meta 被 CLI 替换为真实值（生产环境） | 真实 CLI 后端 | `EventSource(SSE)` + `fetch(REST)` |

**SSE → IncomingEvent 转换**（`sseToIncoming`）：
将 CLI 后端推送的 `DashboardEvent`（kind-based）转换为前端 reducer 可消费的 `IncomingEvent`（type-based）。

**RPC → REST 映射**（`serverRpc`）：
将前端 RPC 命令映射到 CLI 后端 REST API 端点。

### CSS 体系 `styles.css`

- 6800+ 行，完整保留桌面端 oklch 调色板
- 6 套主题：Dark/Graphite、Light/Sandstone、Porcelain、Midnight
- `@media (max-width: 768px)` 移动端自适应层

### 静态资源服务 `assets.ts`

重构 `serveAsset()`：
- 优先使用 `dashboard/dist/app.css`（新 React），fallback 旧 CSS
- 新增 `MIME_MAP` 支持字体/图片等二进制格式
- `loadDistFile()` 支持 `dist/` 和 `dist/assets/` 子目录

---

## 三、修复问题清单

### 1. editMode 字段支持
- **文件**：`src/server/api/settings.ts`
- **问题**：前端模式切换按钮点击后无法传递到后端
- **修复**：新增 `editMode` 字段（review/auto/yolo），写入配置 + 调用 `ctx.setEditMode()`

### 2. 消息重复回显
- **文件**：`dashboard/src/App.tsx` (line 546-549)
- **问题**：前端乐观显示 + SSE 回传同一消息导致重复
- **修复**：busy 状态检测，如果最后一条文本匹配则跳过

### 3. settings 状态完整性
- **文件**：`dashboard/src/lib/tauri-bridge.ts`
- **问题**：settings 保存后缺少 `workspaceDir`、`version` 等字段
- **修复**：合并 `GET /api/settings` + `GET /api/overview` 构建完整 `$settings`

### 4. 会话 REST 路由对齐

| 命令 | 路由 |
| --- | --- |
| `session_list` | `GET /api/sessions` |
| `session_load` | `POST /api/sessions/:name/switch` + `GET /api/sessions/:name` |
| `session_delete` | `DELETE /api/sessions/:name` |
| `new_chat` | `POST /api/sessions/new` |

### 5. SSE 断线重连
- **文件**：`dashboard/src/lib/tauri-bridge.ts`
- Token 通过 query parameter 传递（EventSource 不支持自定义 Header）
- 断线自动重连，指数退避

### 6. 状态栏数据链路
- `assistant_final` 事件新增 `usage` 和 `costUsd` — `src/server/context.ts`
- `handleAssistantFinal` 广播完整 usage — `src/cli/ui/hooks/handle-assistant-final.ts`
- `sseToIncoming` 解析 usage 字段 — `tauri-bridge.ts`

### 7. 静态资源二进制服务
- **文件**：`src/server/assets.ts`
- 字体/图片等服务返回正确 Content-Type
- `loadDistFile()` 查找 `dist/` 和 `dist/assets/` 两个位置

### 8. CSS 构建修复
- **文件**：`dashboard/src/main.tsx`
- **问题**：`styles.css` 未被 JS import，Vite 构建时主题 Token 全部丢失
- **修复**：`main.tsx` 添加 `import "./styles.css"`

### 9. 字体路径修正
- **文件**：`dashboard/vite.config.ts` + `src/server/assets.ts`
- **问题**：CSS `url()` 引用根路径字体文件，服务器无对应路由
- **修复**：字体输出到 `dist/assets/`，服务器支持 `/assets/*` 路由

### 10. 移动端自适应
- **文件**：`dashboard/src/styles.css`（末尾 约 200 行）
- 侧滑抽屉、触控热区（44px）、iOS dvh 适配、全屏模态

### 11. CI 修复（3 轮）
| 轮次 | 问题 | 修复 |
| --- | --- | --- |
| 1 | dashboard node_modules 未安装 | `.github/workflows/ci.yml` 添加 `npm --prefix dashboard ci` |
| 2 | 隐式 `any` 类型错误 | `dashboard/tsconfig.json` 添加 `noImplicitAny: false` |
| 3 | smoke test 在构建前运行 | CI 中 test 和 build 调换顺序 |

---

## 四、验证结果

| 命令 | 结果 |
| --- | --- |
| `npm run build` | ✅ 通过（dashboard + tsup + vendor CSS） |
| `npm run typecheck` | ✅ 通过（root + dashboard） |
| `npm run lint` | ✅ 通过 |
| `npm run test` | ✅ 通过（3374 tests） |
| `npm test -- dashboard-smoke` | ✅ 通过（10/10） |
| CI / build (ubuntu) | ✅ 通过 |
| CI / build (windows) | ✅ 通过 |
| CodeQL | ✅ 通过 |

---

## 五、文件变更清单

### 新增（20+）

| 文件 | 说明 |
| --- | --- |
| `dashboard/package.json` | React 19 + Vite + TypeScript 项目 |
| `dashboard/package-lock.json` | 依赖锁定 |
| `dashboard/vite.config.ts` | Vite 构建配置 + dev-html-rewrite 插件 |
| `dashboard/tsconfig.json` | TS 配置 + paths 映射 |
| `dashboard/index.html` | HTML 入口 |
| `dashboard/src/main.tsx` | 应用入口（import styles.css） |
| `dashboard/src/App.tsx` | 主应用组件 |
| `dashboard/src/styles.css` | 6800+ 行 CSS（主题 + 响应式） |
| `dashboard/src/theme.ts` | oklch 主题管理 |
| `dashboard/src/protocol.ts` | 事件/命令类型 |
| `dashboard/src/lib/tauri-bridge.ts` | 双模式 Tauri API 桥接器 |
| `dashboard/src/ui/*` | 15+ UI 组件 |
| `tests/dashboard-smoke.test.ts` | 构建产物验证 |
| `计划/` | 5 份设计/审查文档 |

### 修改（10+）

| 文件 | 改动 |
| --- | --- |
| `.github/workflows/ci.yml` | dashboard 依赖安装 + 构建顺序修正 |
| `package.json` | 新增 `build:dashboard` |
| `tsup.config.ts` | 删除旧 dashboard entry |
| `src/server/assets.ts` | 通用静态文件 + loadCss 路径修正 |
| `src/server/context.ts` | assistant_final 增加 usage/costUsd |
| `src/server/api/settings.ts` | editMode 字段 |
| `src/cli/ui/hooks/handle-assistant-final.ts` | 广播 usage |
| `tests/server-dashboard.test.ts` | editMode 回归测试 |

### 删除（46）

- 旧 Preact 面板：`dashboard/src/panels/*`（15）+ `dashboard/src/lib/*`（17）+ `dashboard/src/components/*`（1）
- 过期测试：8 个引用已删除模块的测试

---

## 六、延后优化项

| 项目 | 优先级 | 说明 |
| --- | --- | --- |
| app.js 约 900KB 拆包 | P2 | 稳定后做代码分割 |
| 手机端真机调试 | P2 | CSS 已完成，需真实设备验证 |
| SSE 协议标准化 | P2 | 事件转换逻辑下沉后端 |
| Token 安全策略强化 | P2 | 公网部署前需检查 |
| 更新机制清理 | P2 | Web 版由 CLI 管理，不需要桌面 updater |

---

## 七、提 PR 确认清单

- [x] 所有新增文件已 `git add`
- [x] `npm run build` 完整通过
- [x] `npm run typecheck` root + dashboard 通过
- [x] `npm run test` 全量通过
- [x] `npm run lint` 通过
- [x] Rebase 最新 `origin/main`
- [x] 删除 46 个旧文件
- [x] CI 全部绿（ubuntu + windows + CodeQL）
- [x] PR 文档完整（design docs + checklist）
- [x] 分支冲突已解决
