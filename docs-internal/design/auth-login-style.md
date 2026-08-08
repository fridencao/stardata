# 鉴权 / 卡片页统一视觉规范（Auth & Card Style）

> 固化自 `web-admin` 的登录页 `http://localhost/-/welcome/login`（即 `/-/welcome/login` 路由）。
> **所有需要"登录 / 鉴权 / 欢迎"语义的页面，统一参考本规范**，确保品牌一致。
> Keycloak 凭证录入页（`deploy/keycloak/themes/stardata`）必须与本规范保持同步。
>
> **单一事实来源（Single Source of Truth）**：`web-common/src/styles/auth-card.css`
> —— 该文件用 CSS 变量定义了一套 `.auth-card*` 类，承载下方全部尺寸与品牌色。
> `web-admin` 登录页已改用这些类；Keycloak 主题 `stardata.css` 因无法引用 Svelte/Tailwind，
> **镜像了其中的数值**（务必保持同步）。

---

## 1. 页面结构（自上而下）

```
[ 全屏背景 welcome-bg-art.jpg ]
        │
   ┌────┴───────────────┐
   │  ✦  (品牌 logo 方块) │   ← 居中，单独在卡片外
   │  Sign in to StarData │   ← 居中大标题
   └─────────────────────┘
        │
   ┌────┴───────────────┐
   │  Welcome back        │   ← 卡片内标题
   │  Sign in with your   │   ← 卡片内副标题
   │  StarData account…   │
   │  ┌────────────────┐  │
   │  │  Username       │  │   ← 表单字段
   │  └────────────────┘  │
   │  ┌────────────────┐  │
   │  │  Password       │  │
   │  └────────────────┘  │
   │  [   Log in (蓝)   ]  │   ← 主按钮
   └─────────────────────┘
```

- logo + 主标题在**卡片外**居中；卡片内是 "Welcome back" + 副标题 + 表单。
- 背景统一使用 `welcome-bg-art.jpg`（全屏 cover）。

---

## 2. 设计 Token（唯一事实来源）

| 语义 | Tailwind token | 计算值 | 说明 |
|---|---|---|---|
| 品牌主色 | `primary-600` / `accent-primary` | `#155dfc` | logo 方块、主按钮、链接、聚焦边框 |
| 主色 hover | `primary-700` | `#1d4ed8` | 主按钮 hover |
| 标题文字 | `text-fg-accent` | `#1c398e` | "Sign in to StarData" / "Welcome back" |
| 副标题文字 | `text-fg-muted` | `#737373` | 说明性灰字 |
| 卡片背景 | `bg-surface-overlay` | `#ffffff` | 白底卡片 |
| 卡片边框 | `border` | `#e5e5e5` | 1px 浅灰描边 |
| 字段标签 | — | `#334155` / 13px / 600 | 表单 label |
| 字段边框 | — | `#cbd5e1` | 输入框描边 |
| 页面背景 | `welcome-bg-art.jpg` | cover | 与 welcome 布局同源 |
| 字体 | body `font-family` | `-apple-system, BlinkMacSystemFont, "PingFang SC", "SF Pro SC", "SF Pro Text", "Helvetica Neue", sans-serif` | **非** Inter，用系统 UI 栈 |

> `primary-600` 的 oklch 源值：`oklch(0.546 0.245 262.881)` → hex `#155dfc`。

### 字号标度（紧凑，已从构建产物核实）

web-common 的 Tailwind 工具类被**重映射为紧凑值**（已在本仓库打包后的 CSS 核实），下方为实际 px：

| 工具类 | 实际字号 | 行高 | 用途 |
|---|---|---|---|
| `body` | 11px | 1.5 | 基准 / 输入框文字 |
| `text-2xl` | **14px** | 1.35 | 主标题 "Sign in to StarData"（800） |
| `text-base` | **11px** | 1.45 | 卡片标题 "Welcome back"（600，加粗） |
| `text-sm` | **10.5px** | 1.5 | 卡片副标题（非加粗灰字） |
| `text-lg` | 12px | 1.4 | — |
| 主按钮 | **12px** | — | 登录 CTA |

> 这些数值即 `web-common/src/styles/auth-card.css` 中的 `--auth-*` 变量；Keycloak 主题镜像同一组数字。

---

## 3. 元素级规格（px 为换算值，实现用 Tailwind 类优先）

### 3.1 品牌 Logo（`StarDataLogoWordmark`，size=`lg`）
- 方块尺寸：**26 × 26 px**
- 圆角：**8 px**（`rounded-lg`）
- 背景：`#155dfc`
- 内容：白色 `✦`，居中，字号约 16px
- 阴影（可选）：`0 4px 12px rgba(21,93,252,0.30)`

### 3.2 主标题 "Sign in to StarData"
- 字号：**14 px**（`text-2xl`，已重映射为紧凑值）
- 字重：**800**（`font-extrabold`）
- 颜色：`#1c398e`（`text-fg-accent`）
- 对齐：居中

### 3.3 卡片（`.card-pf` / `bg-surface-overlay`）
- 背景：`#ffffff`
- 边框：1px `#e5e5e5`
- 圆角：**6 px**（`rounded-md`）
- 内边距：**24 px**（`p-6`）
- 阴影：`0 10px 30px rgba(15,23,42,0.10)`
- 最大宽度：420 px，水平居中
- 卡片内文字默认左对齐（`text-left`）

### 3.4 卡片内标题 "Welcome back"
- 字号：**11 px**（`text-base`，已重映射为紧凑值）
- 字重：**600**（`font-semibold`）
- 颜色：`#1c398e`
- 对齐：左对齐

### 3.5 卡片内副标题 "Sign in with your StarData account to continue."
- 字号：**10.5 px**（`text-sm`）
- 颜色：`#737373`（`text-fg-muted`）
- 对齐：左对齐
- 与标题紧贴（卡片内同一文本块，仅自然行高，无额外间距）

### 3.6 表单字段
- label：11px / 600 / `#334155`
- 输入框：边框 1px `#cbd5e1`，圆角 6px，内边距 8px 10px，字号 11px，文字 `#1c398e`
- 聚焦：边框 `#155dfc` + `0 0 0 3px rgba(21,93,252,0.15)` 外发光

### 3.7 主按钮（"Log in" / "Log in / Sign up"）
- 背景：`#155dfc`，文字白色（`text-fg-inverse`）
- 圆角：2–4 px（品牌按钮 `rounded-[2px]`，登录表单可用 4px 略柔和）
- 字重：600
- hover：背景 `#1d4ed8`（或 `opacity-80`）
- 尺寸：表单内全宽、内边距 8px 14px、**字号 12px**

### 3.8 链接 / 次要操作
- 颜色：`#155dfc`，hover `#1d4ed8` 加下划线

---

## 4. 校验清单（新增同类页面时自查）

- [ ] 背景为 `welcome-bg-art.jpg` 全屏 cover（或同款浅色 surface）
- [ ] logo 为 26px 蓝方块 `✦`，圆角 8px
- [ ] 主标题 14px / 800 / `#1c398e`
- [ ] 卡片：白底、1px `#e5e5e5` 边框、圆角 6px、内边距 24px、最大宽 420px
- [ ] 卡片内标题 11px / 600；副标题 10.5px / `#737373`；两者均**左对齐**且紧贴
- [ ] 输入框圆角 6px、聚焦蓝色外发光
- [ ] 主按钮 `#155dfc`、白字
- [ ] 字体为系统 UI 栈（非 Inter）

---

## 5. 同步维护

- **web-admin**：规范实例见 `web-admin/src/routes/-/welcome/login/+page.svelte`，logo 组件 `web-common/src/components/icons/StarDataLogoWordmark.svelte`。
- **Keycloak**：主题 `deploy/keycloak/themes/stardata/login/`，品牌样式在 `resources/css/stardata.css`；改规范时此处须同步。
- 改背景图：更新 `welcome-bg-art.jpg`（web-common 与 keycloak 主题各一份）。
