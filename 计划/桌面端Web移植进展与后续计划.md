# DeepSeek-Reasonix 桌面端 Web 移植进展与后续计划

> 面向参与开发/评审的同学：这份文档用于同步当前移植进展、真实启动方式、后续优先级，以及需要大家一起评审和补充建议的地方。

## 一、当前进展

本轮工作目标是将 DeepSeek-Reasonix 桌面端 Tauri + React UI 移植到 CLI 挂载的 Web dashboard 中，替换原有 dashboard 的旧 Preact UI，并为后续移动端兼容打基础。

当前状态：

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| dashboard 独立 Vite + React 项目 | 已完成 | `dashboard/` 已具备独立构建体系 |
| 桌面端 UI 资产迁移 | 已完成 | 已迁移 React 组件、主题、样式、图标、Markdown、代码高亮等 |
| Tauri Web 桥接层 | 已完成基础版本 | `tauri-bridge.ts` 支持 Mock / Server 双模式 |
| 移动端响应式 CSS | 已完成初版 | 已有抽屉、100dvh、触控热区等适配，待真机验证 |
| CLI 真实联调 | **已完成** | REST 路由对齐、SSE 重连、smoke test、构建串联全部完成 |

### 已完成修复清单

#### 阶段 A：桌面端 Web 首屏与 ready 链路 ✅

| 修复项 | 说明 |
| --- | --- |
| 构建产物检测移除 | 移除错误的构建产物检测逻辑（script src 路径不匹配） |
| `$ready` 事件时序 | 确保 `$settings`、`$sessions` 在 `$ready` 之前发出 |

#### 阶段 B：协议与 REST 路由对齐 ✅

| 修复项 | 修改前 | 修改后 |
| --- | --- | --- |
| `session_list` | 直接返回后端数据 | 转换 `mtime` 时间戳为 ISO 字符串 |
| `session_load` | `POST /api/sessions/load?name=...` | 先 `POST /api/sessions/:name/switch` 再 `GET /api/sessions/:name` |
| `session_delete` | `DELETE /api/sessions?name=...` | `DELETE /api/sessions/:name`，删除后自动刷新列表 |
| `new_chat` | 依赖 `data.name` | 调用 `POST /api/sessions/new` 后重新拉取会话列表 |
| `settings_save` | 只 POST 保存，不拉取最新设置 | POST 后合并 GET settings + overview，广播完整 `$settings` |
| `editMode` 切换 | 后端未处理 editMode 字段 | 新增 editMode 写入配置 + 运行时 `setEditMode` 立即生效 |

#### 阶段 C：SSE 与异常状态 ✅

| 修复项 | 说明 |
| --- | --- |
| 断线重连 | 指数退避策略，最多 10 次 |
| Token 过期检测 | 显示"链接已过期，请重新从 CLI 打开" |
| CLI 停止检测 | 超过最大重连次数显示"CLI 已停止，请重新启动" |
| 重连状态提示 | 显示"正在重连…"和"连接已恢复" |
| 提交失败保留草稿 | `/api/submit` 失败时发送错误事件 |
| 用户消息重复显示 | busy 状态下，若最后一条用户消息文本一致，忽略 SSE 回显 |
| settings 保存后 UI 状态被覆盖 | `emitServerSettings` 合并 settings + overview 后广播，避免缺 workspaceDir/version/model 等字段 |

#### 阶段 D：桌面端验收矩阵与自动化 ✅

- 新增 `tests/dashboard-smoke.test.ts`（10 个测试用例）
- 验证构建产物存在性、token 替换、服务器资源加载
- 删除旧 Preact 遗留测试 `tests/dashboard-format.test.ts`
- 新增 editMode 保存、运行时应用、非法值拒绝回归测试

#### 阶段 E：构建与发布整理 ✅

- `package.json` 新增 `build:dashboard` 脚本
- `build` 脚本自动串联 dashboard 构建 → tsup → vendor CSS 复制

#### 阶段 F：状态栏数据链路 ✅

| 修复项 | 说明 |
| --- | --- |
| `assistant_final` 事件 | 新增 `usage` 和 `costUsd` 字段，传递 prompt/completion/cache tokens |
| `handleAssistantFinal` | 广播事件时包含完整 usage 信息 |
| `sseToIncoming` | 解析 `assistant_final` 中的 usage 字段，传递给 `model.final` |
| 余额轮询 | 初始从 overview 获取 balance，之后每 5 秒轮询更新 |
| 缓存命中率 | 通过 `model.final` 事件中的 usage 计算更新 |

重要结论：后续验收应以 **CLI 启动后的真实 Server 模式** 为准，而不是 Vite Mock 模式。

## 二、真实启动方式

当前真实联调流程如下：

```powershell
cd D:\AI\workspace\dashboard
npm run build
cd D:\AI\workspace
npx tsx src/cli/index.ts code --open-dashboard
```

启动后会打开类似下面的地址：

```text
http://127.0.0.1:62197/?token=xxxx
```

说明：

- 端口是动态分配的。
- `token` 是本次 dashboard 的访问令牌。
- 后续调试、验收、截图和问题定位都应优先使用这个 token URL。
- `npm run dev` 的 `http://127.0.0.1:3000` 只用于 Mock 视觉预览，不代表真实 CLI 联调结果。

## 三、主要改动

新增/重构的关键文件：

| 文件 | 说明 |
| --- | --- |
| `dashboard/package.json` | Vite + React 19 + TypeScript 项目配置 |
| `dashboard/vite.config.ts` | dashboard 构建配置，输出 `app.js` / `app.css` |
| `dashboard/src/main.tsx` | Web 入口 |
| `dashboard/src/App.tsx` | 主应用组件，包含多 tab、主题、移动端状态 |
| `dashboard/src/styles.css` | Premium 桌面端样式 + 移动端响应式适配 |
| `dashboard/src/lib/tauri-bridge.ts` | Mock / Server 双模式桥接层 |
| `dashboard/src/protocol.ts` | 前端 IncomingEvent / OutgoingCommand 类型 |
| `dashboard/src/ui/*` | 侧边栏、输入框、线程、卡片、设置、关于等 UI 组件 |
| `src/server/assets.ts` | 静态资源服务，优先加载 dashboard 构建产物 |

已删除/替换：

- 旧的 `dashboard/src/lib/*` Preact 工具层。
- 旧的 `dashboard/src/panels/*` Preact panel 页面。

## 四、当前架构

```text
dashboard React UI
        |
        v
tauri-bridge.ts
        |
        +-- Mock 模式：Vite dev / 视觉预览 / 模拟消息
        |
        +-- Server 模式：CLI token URL / SSE + REST / 真实联调
```

Server 模式核心链路：

- HTML 中由 CLI 注入 `reasonix-mode` 和 `reasonix-token`。
- 前端通过 `/api/events?token=...` 建立 SSE。
- 前端 RPC 命令通过 REST API 转发。
- CLI 后端发送的 `DashboardEvent` 在 `tauri-bridge.ts` 中转换成前端 reducer 可消费的 `IncomingEvent`。

## 五、下一阶段优先级

### 阶段 A：先跑通桌面端 Web

目标：在 Windows 桌面浏览器中，通过 `code --open-dashboard` 打开的 token URL 完成真实 CLI 使用闭环。

验收标准：

- token URL 打开后 3 秒内进入完整 UI，不停留在 `loading…`。
- `tauri-bridge` 识别为 `mode=server`，不是 `mode=mock`。
- 初始 `$tab_opened`、`$settings`、`$sessions`、`$ready` 全部落到当前 tab。
- 输入一条短消息后，用户消息立即显示。
- `assistant_delta`、`assistant_final`、`busy-change` 不丢失、不重复、不乱序。
- 点击停止能调用 `POST /api/abort`，并让 UI 回到可输入状态。
- 覆盖一次工具卡片展示和一次确认/选择/计划弹窗回传。

### 阶段 B：协议与 REST 路由对齐

当前需要重点检查 `tauri-bridge.ts` 中 RPC 到 REST 的映射。

| 前端命令 | 应调用的 REST |
| --- | --- |
| `session_list` | `GET /api/sessions` |
| `session_load` / 切换会话 | `POST /api/sessions/:name/switch` |
| `session_delete` | `DELETE /api/sessions/:name` |
| `new_chat` | `POST /api/sessions/new` 后重新 `GET /api/sessions` |

特别注意：

- `new_chat` 后端当前返回 `{ ok: true }`，前端不应依赖 `data.name`。
- 新建会话成功后应重新拉取会话列表，并根据 `currentSession` 或最新列表刷新 UI。
- POST/DELETE 必须通过 `x-reasonix-token` header 发送 token，不能只依赖 URL query。

### 阶段 C：SSE 与异常状态

需要补齐：

- `/api/events?token=...` 建立后后端不能崩溃。
- 断线重连不能重复创建 turn 或重复追加消息。
- 服务端 ping 只保活，不改变 UI 状态。
- token 无效时显示“链接已过期，请重新从 CLI 打开”。
- SSE 断开时显示“正在重连”，成功后自动恢复。
- 后端退出时显示“CLI 已停止”，并禁用发送。
- `/api/submit` 失败时保留用户草稿。
- 构建产物缺失时 CLI 启动给明确提示，避免白屏。

### 阶段 D：桌面端验收矩阵与自动化

建议引入轻量 smoke test：

```powershell
cd D:\AI\workspace\dashboard
npm run build
cd D:\AI\workspace
npm run typecheck
npm run test -- dashboard server
```

后续可以增加浏览器自动化检查：

- 打开 CLI token URL。
- 断言页面不含 `loading…`。
- 断言 console 无 error。
- 断言侧边栏、主聊天区、输入框存在。

### 阶段 E：构建与发布整理

建议根项目构建自动串联 dashboard：

```json
{
  "build:dashboard": "npm --prefix dashboard run build",
  "build": "npm run build:dashboard && tsup && node scripts/copy-dashboard-vendor-css.mjs"
}
```

原则：

- `dashboard/dist/app.js`、`dashboard/dist/app.css` 只由构建生成，不手写修改。
- `code --open-dashboard` 启动前检查构建产物是否存在。
- Web 端更新由 CLI/npm 管理，不需要桌面 updater。

### 阶段 F：手机端优化（桌面端通过后）

手机端先暂缓，等桌面端 Web 稳定后再做。

后续移动端计划：

- 先用桌面 Chrome/Edge 模拟 390px、430px、768px。
- 再做真机访问。
- 真机不能访问 PC 的 `127.0.0.1`，需要服务端显式绑定 `0.0.0.0` 或 PC 局域网 IP。
- 手机访问形态：`http://<PC局域网IP>:<端口>/?token=<同一令牌>`。
- 覆盖 iPhone Safari、Android Chrome、Windows Edge 窄屏模拟。
- 验证侧滑抽屉、右侧上下文面板、软键盘顶起、滚动惯性、44px 触控热区。

## 六、暂缓项

以下内容建议等桌面端 Web 稳定后再做：

- `app.js` 拆包与首屏性能深度优化。
- 移动端视觉精修。
- 多浏览器细节打磨。
- 高级离线/重连恢复策略。

## 七、希望大家重点给建议的地方

欢迎大家重点评审下面这些问题：

1. 桌面端 Web 的验收标准是否足够完整？
2. `tauri-bridge.ts` 中 RPC 到 REST 的映射是否应该继续放在前端，还是抽一层共享协议？
3. SSE 的事件转换是否应该更靠近后端，减少前端 `sseToIncoming()` 的适配压力？
4. `session_load`、`session_delete`、`new_chat` 的 REST 设计是否需要统一返回格式？
5. token 安全策略是否足够：GET/HTML/SSE 支持 query token，POST/DELETE 强制 header token。
6. 构建流程是否应该由根 `npm run build` 自动构建 dashboard？
7. 桌面端稳定后，移动端优先做布局，还是先做局域网绑定和真机访问链路？

## 八、当前建议执行顺序

1. ✅ 先修桌面端 token URL 首屏与 ready 链路。
2. ✅ 再修 RPC/REST 路由对齐，尤其是 sessions 和 new_chat。
3. ✅ 然后做 SSE 稳定性和异常状态。
4. ✅ 加 smoke test 和构建脚本串联。
5. 桌面端 Web 验收通过后，再进入手机端真机优化。

## 九、验证命令

```powershell
# 1. 类型检查
npm run typecheck

# 2. 构建 dashboard
npm run build:dashboard

# 3. 运行 smoke test
npm run test -- dashboard-smoke

# 4. 完整构建（包含 dashboard）
npm run build

# 5. 启动 CLI 并打开 dashboard
npx tsx src/cli/index.ts code --open-dashboard
```

## 十、下一步：手机端优化（阶段 F）

桌面端 Web 验收通过后，进入手机端优化：

- 先用桌面 Chrome/Edge 模拟 390px、430px、768px。
- 再做真机访问。
- 真机不能访问 PC 的 `127.0.0.1`，需要服务端显式绑定 `0.0.0.0` 或 PC 局域网 IP。
- 手机访问形态：`http://<PC局域网IP>:<端口>/?token=<同一令牌>`。
- 覆盖 iPhone Safari、Android Chrome、Windows Edge 窄屏模拟。
- 验证侧滑抽屉、右侧上下文面板、软键盘顶起、滚动惯性、44px 触控热区。
