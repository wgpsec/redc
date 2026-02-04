# RedC GUI 开发文档

本文档介绍 RedC GUI 的技术架构、开发环境配置和使用说明。

## 技术栈

- **后端框架**: [Wails v2](https://wails.io/) - Go + Web 技术构建跨平台桌面应用
- **前端框架**: [Svelte](https://svelte.dev/) - 轻量级响应式 UI 框架
- **UI 组件**: [TailwindCSS](https://tailwindcss.com/) + [DaisyUI](https://daisyui.com/)
- **构建工具**: Vite

## 目录结构

```
├── app.go              # Wails 后端 API 实现
├── main.go             # GUI 入口（Wails 应用配置）
├── cmd/cli/main.go     # CLI 入口（独立）
├── frontend/           # 前端代码
│   ├── src/
│   │   └── App.svelte  # 主界面组件
│   ├── wailsjs/        # Wails 自动生成的 JS 绑定
│   ├── index.html
│   ├── package.json
│   └── tailwind.config.js
├── build/
│   └── bin/            # 构建产物
│       └── redc-gui.app/
└── wails.json          # Wails 项目配置
```

## 环境配置

### 前置依赖

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 检查环境
wails doctor
```

### 安装前端依赖

```bash
cd frontend
npm install
```

## 开发模式

```bash
# 在项目根目录运行开发服务器（支持热重载）
wails dev
```

## 构建发布

```bash
# 构建生产版本
wails build

# 产物位置
# macOS: build/bin/redc-gui.app
# Windows: build/bin/redc-gui.exe
# Linux: build/bin/redc-gui
```

## 后端 API (app.go)

### 主要方法

| 方法 | 说明 |
|------|------|
| `ListCases()` | 获取所有场景列表 |
| `ListTemplates()` | 获取可用模板列表 |
| `StartCase(caseID)` | 启动指定场景 |
| `StopCase(caseID)` | 停止指定场景 |
| `RemoveCase(caseID)` | 删除指定场景 |
| `CreateCase(template, name, vars)` | 创建新场景（异步） |
| `GetCaseOutputs(caseID)` | 获取场景的 Terraform outputs |
| `GetConfig()` | 获取配置信息 |

### 事件通信

后端通过 Wails Events 与前端通信：

```go
// 发送日志到前端
runtime.EventsEmit(a.ctx, "log", message)

// 通知前端刷新数据
runtime.EventsEmit(a.ctx, "refresh", nil)
```

前端监听事件：

```javascript
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js';

EventsOn('log', (message) => {
  // 处理日志
});

EventsOn('refresh', async () => {
  // 刷新数据
});
```

## 前端组件 (App.svelte)

### 页面结构

- **仪表盘 (Dashboard)**: 场景列表、创建场景、操作按钮
- **控制台 (Console)**: 实时日志输出
- **设置 (Settings)**: 配置信息展示

### 状态管理

```javascript
const stateConfig = {
  'running': { label: '运行中', color: 'text-emerald-600' },
  'stopped': { label: '已停止', color: 'text-slate-500' },
  'error': { label: '异常', color: 'text-red-600' },
  'created': { label: '已创建', color: 'text-blue-600' },
  'starting': { label: '启动中', color: 'text-amber-600' },
  'stopping': { label: '停止中', color: 'text-amber-600' },
  'removing': { label: '删除中', color: 'text-amber-600' }
};
```

### 关键功能

1. **异步操作**: 启动/停止/删除操作在后台 goroutine 执行，不阻塞 UI
2. **中间状态**: 操作进行时显示"启动中"等状态，防止重复点击
3. **自动刷新**: 操作完成后自动刷新仪表盘数据
4. **展开详情**: 点击场景行可展开查看 Terraform outputs
5. **复制功能**: outputs 支持一键复制

## 配置路径

GUI 与 CLI 共享相同的配置路径逻辑：

1. 优先检查当前目录下的 `.redc` 配置
2. 其次检查用户目录 `~/redc/` 

默认路径：
- 配置目录: `~/redc/`
- 模板目录: `~/redc/redc-templates/`
- 任务结果: `~/redc/task-result/`

## 窗口配置

在 `main.go` 中配置窗口参数：

```go
wails.Run(&options.App{
    Title:  "RedC - 红队基础设施管理",
    Width:  1440,
    Height: 900,
    // ...
})
```

## 常见问题

### 1. 创建场景时 GUI 卡住

**原因**: `CaseCreate` 中的 `TfPlan` 是耗时操作
**解决**: 已将 `CreateCase` 改为异步执行

### 2. Terraform 版本不匹配

**错误**: `plan files cannot be transferred between different Terraform versions`
**解决**: GUI 使用与 CLI 相同的 terraform-exec 库，确保版本一致

### 3. 操作后状态未更新

**解决**: 操作完成后会发送 `refresh` 事件，前端自动刷新

## 开发注意事项

1. **并发安全**: `App` 结构体使用 `sync.Mutex` 保护共享状态
2. **Goroutine**: 耗时操作（如 Terraform Apply）在 goroutine 中运行
3. **错误恢复**: 使用 `defer recover()` 防止 panic 导致应用崩溃
4. **工作目录**: 注意 Wails 运行时的 CWD 问题
