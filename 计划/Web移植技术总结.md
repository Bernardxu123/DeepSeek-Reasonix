# DeepSeek-Reasonix 桌面端 Web 移植 — 完整技术总结

> 生成时间：2026-05-20  
> 关联文档：`implementation_plan.md` · `task.md` · `Web移植PR审查清单.md`

---

## 一、项目概述

将 **DeepSeek-Reasonix 桌面端 (Tauri + React + CSS)** 的 premium UI 移植到 **Web 端 (CLI Dashboard)**，放弃旧 Preact + HTM 方案，并实现移动端自适应。

采用**渐进式两阶段开发**：

| 阶段 | 内容 | 状态 |
| --- | --- | --- |
| 一 | 建立 Vite + React 独立开发环境 + Tauri 桥接器 + Mock 数据流 | ✅ 完成 |
| 二 | CLI SSE/REST 真实桥接 + 生产构建 + 联调 | ✅ 完成 |

---

## 二、架构图

```
┌──────────────────────────────────────────┐
│  浏览器 (Web / Mobile / Tablet)          │
│  ┌────────────────────────────────────┐  │
│  │  dashboard/dist/app.js (React 19)  │  │
│  │  ┌──────────────────────────────┐  │  │
│  │  │  tauri-bridge.ts (双模式)     │  │  │
│  │  │                             │  │  │
│  │  │  Mock 模式 ←── Vite 开发    │  │  │
│  │  │  Server 模式 ←── CLI 生产    │  │  │
│  │  │         │                   │  │  │
│  │  │    ┌────┴────┐              │  │  │
│  │  │    │ SSE/REST │              │  │  │
│  │  │    │ 转换层   │              │  │  │
│  │  │    └────┬────┘              │  │  │
│  │  └─────────┼───────────────────┘  │  │
│  └────────────┼──────────────────────┘  │
│               │  HTTP (SSE + REST API)  │
└───────────────┼─────────────────────────┘
                │
┌───────────────┼─────────────────────────┐
│  CLI 后端     │                         │
│  ┌────────────┴──────────────────────┐  │
│  │  src/server/ (HTTP Server)        │  │
│  │  ├─ /api/events  (SSE 事件流)     │  │
│  │  ├─ /api/*       (REST API)      │  │
│  │  └─ /assets/*    (静态资源)       │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  CacheFirstLoop (CLI 内核)        │  │
│  └───────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

---

## 三、技术栈

| 层级 | 技术 |
| --- | --- |
| 前端框架 | React 19 + TypeScript |
| 构建工具 | Vite 5 (开发) / tsup (CLI) |
| CSS 配色 | oklch 色彩空间，6 套主题 |
| Markdown | react-markdown + KaTeX + Prism |
| 字体 | Geist + Geist Mono + Inter |
| 虚拟滚动 | react-virtuoso |
| 图标 | lucide-react + 自定义 SVG |
| 协议 | SSE (推) + REST (拉) |
| 双模式 | Mock (Vite 开发) ↔ Server (CLI 生产) |

---

## 四、核心模块说明

### 4.1 tauri-bridge.ts — 双模式桥接器

| 模式 | 触发条件 | 数据来源 | 通信方式 |
| --- | --- | --- | --- |
| **Mock** | `<meta reasonix-mode>` = `__REASONIX_MODE__` (Vite) | 内置模拟数据 | 直接 `broadcast()` |
| **Server** | meta 被 CLI 替换为真实值 | CLI 后端 | `EventSource(SSE)` + `fetch(REST)` |

**事件转换** (`sseToIncoming`)：
- SSE `DashboardEvent` (kind-based: `assistant_delta`, `tool_start`, `modal-up`)  
  → 前端 `IncomingEvent` (type-based: `model.delta`, `tool.preparing`, `$confirm_required`)

**命令映射** (`serverRpc`)：

| 前端 RPC | REST API |
| --- | --- |
| `user_input` | `POST /api/submit` |
| `abort` | `POST /api/abort` |
| `session_list` | `GET /api/sessions` |
| `session_load` | `POST /api/sessions/:name/switch` + `GET /api/sessions/:name` |
| `session_delete` | `DELETE /api/sessions/:name` |
| `new_chat` | `POST /api/sessions/new` + 重新拉取列表 |
| `settings_save` | `POST /api/settings` + 重新拉取 overview |
| `confirm_*` | `POST /api/modal` |
| `jobs_list` | `GET /api/usage` |
| `mention_query` | `GET /api/files/search` |

### 4.2 assets.ts — 服务端静态资源

修改点：
- `loadCss()` 优先加载 `dashboard/dist/app.css`（新 React CSS），fallback 到 `dashboard/app.css`（旧 Preact CSS）
- 新增 `loadDistFile()` 支持 `dist/` 和 `dist/assets/` 两个子目录
- 新增 `MIME_MAP` 通用 MIME 类型表，支持字体 (.woff2/.woff/.ttf)、图片 (.svg/.png/.ico)
- 新增 `loadCachedBinary()` / `binaryCache` 分离文本/二进制缓存

### 4.3 styles.css — CSS 体系

- 总行数：6806 行
- 主题 Token：`:root` / `[data-theme="dark"]` / `[data-theme="light"]` / `[data-theme-style="graphite"]` / `midnight` / `porcelain` / `sandstone`
- oklch 调色板：bg, fg, accent, success, warning, danger, violet 等 30+ 自定义属性
- 移动端：`@media (max-width: 768px)` 完整断点，含侧滑抽屉、触控热区、dvh 适配

### 4.4 移动端自适应

| 能力 | 实现 |
| --- | --- |
| 单栏布局 | `grid-template-columns: 1fr` |
| 侧滑抽屉 | `position:fixed` + `translate3d(-100%,0,0)` + `cubic-bezier` 缓动 |
| 暗色遮罩 | `.mobile-overlay` + `backdrop-filter: blur(2px)` |
| iOS 键盘 | `100dvh` 视口 + TextArea `font-size:16px` |
| 触控热区 | 所有交互元素 `min-height:44px` |
| OS 控件 | `html[data-web="true"] .win-controls` `display:none` |
| 原生动量 | `-webkit-overflow-scrolling:touch` |
| 全屏模态 | `width:100vw; height:100dvh` |

---

## 五、改动文件汇总

### 新增文件 (17+ UI 组件)

| 文件 | 说明 |
| --- | --- |
| `dashboard/package.json` | 项目清单 |
| `dashboard/vite.config.ts` | Vite + `dev-html-rewrite` 插件 + 字体输出路径 |
| `dashboard/tsconfig.json` | 含 `@tauri-apps/*` → `tauri-bridge.ts` 路径映射 |
| `dashboard/src/main.tsx` | 入口，import `styles.css` + 字体 |
| `dashboard/src/App.tsx` | 主应用 (`TabRuntime` 多标签 + `mobileSideOpen` 状态) |
| `dashboard/src/styles.css` | 6806 行完整 CSS (暗色主题 + 移动端) |
| `dashboard/src/theme.ts` | oklch 调色板 + 主题切换 |
| `dashboard/src/protocol.ts` | 事件/命令类型定义 |
| `dashboard/src/lib/tauri-bridge.ts` | **核心** — 双模式 Tauri API 桥接 |
| `dashboard/src/ui/*` | 15+ UI 组件 (侧栏/输入框/线程/卡片/设置等) |
| `tests/dashboard-smoke.test.ts` | 构建产物验证测试 |

### 删除文件 (46)

| 类型 | 数量 | 说明 |
| --- | --- | --- |
| 旧 Preact 面板 | 38 | `src/panels/*` + `src/lib/*` |
| 过期测试 | 8 | 引用已删除模块的测试 |

### 修改文件

| 文件 | 改动 |
| --- | --- |
| `src/server/assets.ts` | 通用静态文件服务、二进制缓存、`loadCss` 路径修正 |
| `src/server/context.ts` | `assistant_final` 新增 `usage`/`costUsd` |
| `src/server/api/settings.ts` | 新增 `editMode` 字段 |
| `src/cli/ui/hooks/handle-assistant-final.ts` | 广播事件携带 usage |
| `tsup.config.ts` | 删除旧 dashboard entry |
| `package.json` | 新增 `build:dashboard` |
| `dashboard/index.html` | 生产路径 + 增强 viewport meta |
| `dashboard/tsconfig.json` | 路径映射 + TS 配置 |
| `dashboard/src/i18n/*` | 语言文件更新 |
| `dashboard/src/ui/useAutoScroll.ts` | 注释清理 |
| `tests/server-dashboard.test.ts` | editMode 回归测试 |

---

## 六、开发与验证命令

```bash
# ── Dashboard 独立开发 ──
cd D:\AI\workspace\dashboard
npm run dev          # Vite Mock 开发 (http://127.0.0.1:3000)
npm run build        # 生产构建 → dist/app.js + dist/app.css
npm run dev -- --host 0.0.0.0  # 局域网手机测试

# ── CLI 联调 ──
cd D:\AI\workspace
npm run build        # 完整构建 (dashboard → tsup → vendor CSS)
npx tsx src/cli/index.ts code --open-dashboard  # 启动 + 浏览器

# ── 验证 ──
npm run typecheck    # TypeScript 编译检查
npm run test         # 全量测试 (229 files / 3271 tests)
npm run lint         # Biome 代码检查
```

### 验证通过项

| 命令 | 结果 |
| --- | --- |
| `npm run build` | ✅ 通过 |
| `npm run typecheck` | ✅ 通过 |
| `npm run lint` | ✅ 通过 |
| `npm test -- dashboard-smoke` | ✅ 10/10 |
| `npm run test` | ✅ 229/3271 |

---

## 七、后续优化建议

| 项目 | 优先级 | 说明 |
| --- | --- | --- |
| 手机端真机调试 | P2 | CSS 已完成，需连接手机验证触控体验 |
| app.js 拆包优化 | P2 | 当前约 900KB，可按路由/功能拆分 |
| SSE 协议标准化 | P2 | 将事件转换逻辑下沉到后端，统一格式 |
| Token 安全策略 | P2 | 公网部署前需检查 CSRF 防护 |
| 移动端右面板支持 | 长期 | 当前仅左侧抽屉，右侧上下文面板待设计 |
