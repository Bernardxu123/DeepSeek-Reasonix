# 任务清单: DeepSeek-Reasonix 桌面 UI 移植与移动端自适应

- `[ ]` **阶段一：构建 `dashboard` 下的独立 Vite + React + TS 项目环境**
  - `[ ]` 创建 `dashboard/package.json`，配置 Vite、React 19 和 TypeScript
  - `[ ]` 创建 `dashboard/tsconfig.json` 配置 TS 编译选项
  - `[ ]` 创建 `dashboard/vite.config.ts` 并配置 Rollup 输出固定文件名 `app.js` 与 `app.css`

- `[ ]` **阶段二：迁移桌面端 UI 资产并实装 Tauri API 兼容层**
  - `[ ]` 拷贝 `desktop/src` 中的 React 核心组件、图标、i18n 配置和纯 CSS 到 `dashboard` 目录
  - `[ ]` 编写 `dashboard/src/lib/tauri-bridge.ts` API 兼容层与** Mock 协议模拟器**，接管所有 `invoke` 调用并返回带有推理打字、工具调用的模拟事件流

- `[ ]` **阶段三：手机端（移动端）自适应 UI 精细重构**
  - `[ ]` 在 `dashboard/src/styles.css` 中设计一整套 `@media (max-width: 768px)` 响应式布局方案
  - `[ ]` 在 `dashboard/src/App.tsx` 中移除 Tauri OS 级别控制器，并为手机端添加“☰ 会话菜单/侧滑抽屉”的交互机制
  - `[ ]` 精调 TextArea 虚拟键盘表现、锁定移动端防拉伸页面，以及优化触控热区为至少 `44px * 44px`

- `[ ]` **阶段四：阶段效果实机验证与 CLI 真实联调**
  - `[ ]` 运行 Vite 服务，分别使用 PC 浏览器、Chrome 移动端模拟器和真实手机局域网测试 Mock 状态下的视觉表现
  - `[ ]` 结合 CLI WebSocket 双向通信机制，实装生产环境构建并与 CLI 后端真实调优
