# CLI UI 重构方案 B: 混合架构实施计划

## 架构概述

采用 "React 负责决策，Rust 负责像素" 的混合架构：

```
┌─────────────────────────────────────────────────────────────┐
│                     React (Ink) 决策层                        │
│  - 状态管理 (useState/useReducer)                              │
│  - 业务逻辑                                                    │
│  - 卡片组件树                                                  │
│  - 布局计算 (Yoga)                                             │
└──────────────────┬──────────────────────────────────────────┘
                   │ JSON Frame Protocol (stdin)
                   ▼
┌─────────────────────────────────────────────────────────────┐
│                  Rust 渲染层 (reasonix-render)                │
│  - 双缓冲帧缓冲                                               │
│  - Dirty Rect 追踪                                           │
│  - Crossterm/Ratatui 绘制                                    │
│  - 增量渲染优化                                               │
└──────────────────┬──────────────────────────────────────────┘
                   │ ANSI Escape Codes (stdout)
                   ▼
              终端显示器
```

## 核心问题诊断

### 当前抖动原因
1. **React 重渲染链式反应**: App.tsx 78+ useState/useEffect
2. **Yoga 布局瓶颈**: 每次卡片高度变化触发完整布局重算
3. **Ink reconciler 开销**: 微小变化也走完整 React diff
4. **useBoxMetrics 测量振荡**: CardStream 中测量触发连锁更新

## 实施阶段

### Phase 1: 架构解耦 (1-2 周)

#### 1.1 创建 Frame Buffer 中间层
文件：`src/cli/ui/frame-buffer.ts`
- 离屏帧缓冲，存储 Cell 数组
- 脏矩形追踪
- 固定频率刷新 (60fps cap)

#### 1.2 自定义 Ink Primitive
文件：`src/cli/ui/primitives/RustBackedBox.tsx`
- 替代 Ink 的 Box/Text 组件
- 直接写入帧缓冲，跳过 Yoga
- 使用 `useLayoutEffect` 批量提交

#### 1.3 Rust 渲染器增强
文件：`crates/reasonix-render/src/frame_protocol.rs` ✅ 已创建
- JSON 协议定义
- 脏矩形合并逻辑
- 双缓冲实现

### Phase 2: 布局引擎优化 (1 周)

#### 2.1 虚拟列表实现
修改：`src/cli/ui/layout/CardStream.tsx`
- 基于行的虚拟化
- 预加载相邻卡片
- 固定高度占位符

#### 2.2 增量包裹优化
修改：`src/cli/ui/cards/useIncrementalWrap.ts`
- 添加 debounce (32ms)
- requestIdleCallback 批处理
- 超长文本截断策略

### Phase 3: Rust 后端接入 (1-2 周)

#### 3.1 NAPI 绑定
文件：`crates/reasonix-render/src/napi.rs`
- Node.js 原生模块
- 零拷贝帧传输

#### 3.2 性能基准测试
文件：`benchmarks/cli-render-bench.ts`
- 对比 Ink vs Rust 渲染延迟
- CPU/内存占用分析

## 预期收益

| 指标 | 当前 | 目标 | 改进 |
|------|------|------|------|
| 渲染延迟 | 50-100ms | <8ms | 85%↓ |
| CPU 占用 | 15-25% | <5% | 80%↓ |
| 内存峰值 | 200MB | <80MB | 60%↓ |
| 抖动频率 | 高 | 无 | 100%↓ |

## 回滚策略

每个 Phase 独立可回滚：
1. Feature flag 控制开关
2. A/B 测试验证
3. 性能回归自动检测

## 下一步行动

1. ✅ 创建 Frame Protocol 定义 (Rust + TS)
2. ⏳ 实现 FrameBuffer 类
3. ⏳ 创建 RustBackedBox 组件
4. ⏳ 修改 CardStream 使用新架构
5. ⏳ 性能测试验证
