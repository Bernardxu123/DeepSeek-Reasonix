# DeepSeek-Reasonix 桌面端 Web 移植 PR 审查清单

> 本文档用于代码审查，列出了本次 Web 移植的所有变更、修复的问题点及对应代码位置。

---

## 当前阻塞项（提 PR 前必须解决）

| 级别 | 问题 | 状态 | 说明 |
| --- | --- | --- | --- |
| ~~P0~~ | 根构建失败 `npm run build` | ✅ 已修复 | 删除 `tsup.config.ts` 中旧 dashboard entry，Vite 产物成为唯一构建链 |
| ~~P0~~ | 新增文件未进入 git 跟踪 | ✅ 已 stage | 所有新增 React Dashboard 文件已纳入 git |
| ~~P1~~ | 静态资源服务破坏二进制资源 | ✅ 已修复 | `serveAsset()` 区分文本/二进制，字体/图片用 Buffer 服务 |
| ~~P1~~ | 本地分支落后 origin/main 23 个提交 | ✅ 已 rebase | 已同步最新 origin/main |
| ~~P1~~ | 过期测试引用已删除模块 | ✅ 已清理 | 删除 7 个旧 Preact 测试 + 1 个 dashboard-format 测试 |
| ~~P2~~ | smoke test 覆盖不了真实构建链 | ✅ 已记录 | PR 清单把 `npm run build` 提升为必跑项 |

**整体状态：✅ 可以准备提交**

---

## 一、变更概览

| 类别 | 文件数 | 说明 |
| --- | --- | --- |
| 删除旧 Preact 面板 | 38 files | `dashboard/src/panels/*`、`dashboard/src/lib/*` 等旧文件 |
| 删除过期测试 | 8 files | `dashboard-format` + 7 个引用旧模块的测试 |
| 新增 React Dashboard | 17 files | `dashboard/src/App.tsx`、`tauri-bridge.ts`、`ui/*` 等 |
| 修改核心文件 | 10 files | 后端路由、SSE 事件、设置 API、tsup 配置、测试等 |

---

## 二、修复问题清单（重点审查）

### 1. review / auto / yolo 按钮无法真正切换

**问题**：前端点击模式按钮后，后端 POST /api/settings 之前没有处理 `editMode` 字段。

**修复**：
- `src/server/api/settings.ts` (line 147-153, 233-237)
  - 新增 `editMode` 字段验证（review | auto | yolo）
  - 写入配置文件
  - 调用运行时 `ctx.setEditMode(mode)` 立即生效，无回调时降级到 `saveEditMode`

```typescript
if (fields.editMode !== undefined) {
  if (typeof fields.editMode !== "string" || !VALID_EDIT_MODES.has(fields.editMode)) {
    return { status: 400, body: { error: "editMode must be review | auto | yolo" } };
  }
  cfg.editMode = fields.editMode as EditMode;
  changed.push("editMode");
}
// ...
if (fields.editMode !== undefined) {
  const mode = fields.editMode as EditMode;
  if (ctx.setEditMode) ctx.setEditMode(mode);
  else saveEditMode(mode, ctx.configPath);
}
```

### 2. 发送消息在网页显示两次

**问题**：前端本地先乐观显示用户消息，SSE 又从后端回传同一条 `user.message`，导致重复。

**修复**：
- `dashboard/src/App.tsx` (line 546-549)
  - busy 状态下，如果最后一条用户消息文本与 SSE 回显一致，忽略回显

```typescript
case "user.message": {
  const last = state.messages[state.messages.length - 1];
  if (state.busy && last?.kind === "user" && last.text === ev.text) {
    return state; // 忽略重复回显
  }
  // ... 正常处理
}
```

### 3. settings 保存后 UI 状态可能被半截 $settings 覆盖

**问题**：`settings_save` / `settings_get` 只返回 settings 数据，缺少 `workspaceDir`、`version`、`model` 等字段（这些在 overview 中）。

**修复**：
- `dashboard/src/lib/tauri-bridge.ts` (line 331-347, 531-541)
  - 新增 `emitServerSettings` 函数，合并 settings + overview 后广播完整 `$settings`
  - `settings_save` 后同时拉取 settings 和 overview，确保 UI 状态完整

```typescript
function emitServerSettings(settings: any, overview?: any): void {
  emitEvent({
    type: "$settings",
    tabId: "tab-1",
    reasoningEffort: settings?.reasoningEffort ?? overview?.reasoningEffort ?? "high",
    editMode: settings?.editMode ?? overview?.editMode ?? "review",
    budgetUsd: settings?.budgetUsd ?? overview?.budgetUsd ?? null,
    workspaceDir: overview?.cwd ?? "",
    model: overview?.model ?? settings?.model ?? "deepseek-reasoner",
    preset: settings?.preset ?? overview?.preset ?? "auto",
    version: overview?.version ?? "",
    // ... 其他字段
  });
}
```

### 4. 会话操作 REST 路由对齐

| 命令 | 修改前 | 修改后 |
| --- | --- | --- |
| `session_list` | 直接返回后端数据 | 转换 `mtime` 时间戳为 ISO 字符串 |
| `session_load` | `POST /api/sessions/load?name=...` | 先 `POST /api/sessions/:name/switch` 再 `GET /api/sessions/:name` |
| `session_delete` | `DELETE /api/sessions?name=...` | `DELETE /api/sessions/:name`，删除后自动刷新列表 |
| `new_chat` | 依赖 `data.name`（后端返回 `{ ok: true }`） | 创建后重新拉取会话列表 |

**位置**：`dashboard/src/lib/tauri-bridge.ts` (line 470-520)

### 5. SSE 断线重连与异常状态

| 修复项 | 说明 | 位置 |
| --- | --- | --- |
| 断线重连 | 指数退避策略，最多 10 次 | `tauri-bridge.ts` (line 250-275) |
| Token 过期检测 | 显示"链接已过期，请重新从 CLI 打开" | `tauri-bridge.ts` (line 233-239) |
| CLI 停止检测 | 超过最大重连次数显示"CLI 已停止" | `tauri-bridge.ts` (line 257-264) |
| 重连状态提示 | "正在重连…" / "连接已恢复" | `tauri-bridge.ts` (line 43-55) |

### 6. 状态栏数据链路（缓存/Tokens/金额/余额）

| 修复项 | 说明 | 位置 |
| --- | --- | --- |
| `assistant_final` 事件 | 新增 `usage` 和 `costUsd` 字段 | `src/server/context.ts` (line 240-251) |
| `handleAssistantFinal` | 广播事件时包含完整 usage 信息 | `src/cli/ui/hooks/handle-assistant-final.ts` (line 56-72) |
| `sseToIncoming` | 解析 `assistant_final` 中的 usage 字段 | `tauri-bridge.ts` (line 91-102) |
| 余额轮询 | 初始从 overview 获取，之后每 5 秒轮询 | `tauri-bridge.ts` (line 280-320) |

### 7. 静态资源二进制服务修复

**问题**：`serveAsset()` 支持 `.woff/.png/.ico`，但 `loadCachedFile()` 把所有文件都 `buf.toString("utf8")` 后返回。字体/图片会被当文本发送，浏览器加载会损坏。

**修复**：
- `src/server/assets.ts`
  - 新增 `binaryCache` 分离文本/二进制缓存
  - 新增 `loadCachedBinary()` 返回 Buffer
  - `BINARY_EXTS` 集合标识二进制扩展名（`.woff2`, `.woff`, `.ttf`, `.png`, `.ico`）
  - `serveAsset()` 返回类型改为 `{ body: string | Buffer; contentType: string }`
  - `res.end()` 原生支持 string 和 Buffer，无需修改调用方

---

## 三、新增测试

### 1. Smoke Test
- **文件**：`tests/dashboard-smoke.test.ts`
- **用例**：10 个
- **覆盖**：构建产物存在性、token 替换、服务器资源加载

### 2. editMode 回归测试
- **文件**：`tests/server-dashboard.test.ts` (line 1126-1157)
- **用例**：
  - `POST /api/settings persists and applies editMode` — 验证写入配置 + 运行时生效
  - `POST /api/settings rejects invalid editMode` — 验证非法值拒绝

### 3. 删除旧测试
- `tests/dashboard-format.test.ts` — 引用已删除的 `dashboard/src/lib/format.js`
- `tests/dashboard-budget.test.ts` — 引用已删除的 `dashboard/src/lib/budget`
- `tests/dashboard-bus-filter.test.ts` — 引用已删除的 `dashboard/src/lib/bus-filter`
- `tests/dashboard-loop-control.test.ts` — 引用已删除的 `dashboard/src/lib/loop-control`
- `tests/dashboard-mcp-spec.test.ts` — 引用已删除的 `dashboard/src/lib/mcp-spec`
- `tests/dashboard-version.test.ts` — 引用已删除的 `dashboard/src/lib/version`
- `tests/diff-parser.test.ts` — 引用已删除的 `dashboard/src/lib/diff-parser`
- `tests/semantic-panel.test.ts` — 引用已删除的 `dashboard/src/panels/semantic`

---

## 四、构建流程变更

### package.json
```json
{
  "scripts": {
    "build:dashboard": "npm --prefix dashboard run build",
    "build": "npm run build:dashboard && tsup && node scripts/copy-dashboard-vendor-css.mjs"
  }
}
```

### 构建产物
- `dashboard/dist/app.js` — 910 KB（minified）
- `dashboard/dist/app.css` — 257 KB

---

## 九、复杂项（可延后）

| 项目 | 是否阻塞 PR | 建议时机 |
| --- | --- | --- |
| app.js 体积约 900KB 拆包优化 | 不阻塞 | 桌面 Web 稳定后做 |
| 手机端布局优化 | 不阻塞 | 桌面端 PR 合并后单独做 |
| tauri-bridge.ts RPC 到 REST 的协议层抽象 | 不阻塞 | 后续重构 PR |
| SSE 事件转换下沉到后端 | 不阻塞 | 等事件协议稳定后做 |
| Token query/header 安全策略强化 | 轻微阻塞 | 公网/远程访问前必须做，本地 127.0.0.1 可先记录风险 |

---

## 六、验证命令

```powershell
# 1. 类型检查
npm run typecheck

# 2. 完整构建（包含 dashboard）— 必跑项
npm run build

# 3. 运行 smoke test
npm run test -- dashboard-smoke

# 4. 运行完整测试
npm run test

# 5. 启动 CLI 并打开 dashboard
npx tsx src/cli/index.ts code --open-dashboard
```

### 验证结果

| 命令 | 结果 |
| --- | --- |
| `npm run build` | ✅ 通过 |
| `npm run typecheck` | ✅ 通过 |
| `npm run lint` | ✅ 通过（1 个既有 warning） |
| `npm run test -- dashboard-smoke` | ✅ 通过（10 tests） |
| `npm run test` | ✅ 通过（229 files / 3271 tests） |

---

## 七、文件变更清单

> 以下基于 `git status` 整理，提 PR 前需用 `git diff --name-status --cached` 重新生成。

### 新增文件（17）
```
dashboard/package.json
dashboard/vite.config.ts
dashboard/src/main.tsx
dashboard/src/App.tsx
dashboard/src/protocol.ts
dashboard/src/lib/tauri-bridge.ts
dashboard/src/styles.css
dashboard/src/theme.ts
dashboard/src/icons.tsx
dashboard/src/CodeView.tsx
dashboard/src/CommandPalette.tsx
dashboard/src/Markdown.tsx
dashboard/src/qq-settings.ts
dashboard/src/vite-env.d.ts
dashboard/src/ui/ (侧边栏、输入框、线程、卡片、设置等组件)
tests/dashboard-smoke.test.ts
```

### 删除文件（46）
```
# 旧 Preact 面板（38）
dashboard/src/components/chat-internals.ts
dashboard/src/lib/{api,budget,bus-filter,bus,diff-parser,error-boundary,file-tree,format,html,i18n,line-comments,loop-control,markdown,mcp-spec,review-diffs,use-poll,version}.ts
dashboard/src/panels/{changes,chat,hooks,mcp,memory,overview,permissions,plans,semantic,sessions,settings,skills,system,tools,usage}.ts

# 过期测试（8）
tests/dashboard-format.test.ts
tests/dashboard-budget.test.ts
tests/dashboard-bus-filter.test.ts
tests/dashboard-loop-control.test.ts
tests/dashboard-mcp-spec.test.ts
tests/dashboard-version.test.ts
tests/diff-parser.test.ts
tests/semantic-panel.test.ts
```

### 修改文件（10）
```
dashboard/index.html
dashboard/tsconfig.json
dashboard/src/i18n/{en,index,zh-CN}.ts
dashboard/src/lib/tauri-bridge.ts (横幅注释清理)
dashboard/src/ui/useAutoScroll.ts (长块注释压短)
package.json
tsup.config.ts (删除旧 dashboard entry)
src/server/context.ts
src/server/api/settings.ts
src/server/assets.ts (二进制资源服务修复 + 格式)
src/cli/ui/hooks/handle-assistant-final.ts
tests/server-dashboard.test.ts
```

---

## 八、提 PR 前必须确认

- [x] 新增 React Dashboard 文件全部 `git add`
- [x] `dashboard/package.json`、`dashboard/package-lock.json` 已 stage
- [x] `dashboard/vite.config.ts` 已 stage
- [x] `tests/dashboard-smoke.test.ts` 已 stage
- [x] `计划/` 文档已 stage
- [x] `npm run build` 通过
- [x] `npm run typecheck` 通过
- [x] `npm run test` 通过（229 files / 3271 tests）
- [x] rebase 最新 `origin/main`
- [x] 删除 8 个过期测试文件
- [x] 清理 tauri-bridge.ts 横幅注释
- [x] 压短 useAutoScroll.ts 长块注释
- [x] 修复 assets.ts 格式问题
