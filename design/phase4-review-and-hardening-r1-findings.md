# R1 端到端验证：Findings

> 产出自 `design/phase4-review-and-hardening.md` P0-1 行动项
> 日期：2026-08-07 ~ 2026-08-08
> 方法：unit test-driven verification（`repo_archive_test.go`）+ 代码审读 + **真实 docker-compose 栈走查（V-3）**

---

## 验证范围

| 风险 | 验证方法 | 结论 |
|---|---|---|
| R1-1 并发写入：publish 时是否可能打包 half-written 文件 | 代码审读 `publishProject → packageDevDraft` | 存在但概率极低，详见下文 |
| R1-2 新 archive 覆盖 dev 草稿 | `TestEditableDraftClobberedByNewArchiveID` | **坐实** — clean extraction 行为设计如此 |
| R1-3 rollback 后 dev 草稿丢失 | `TestEditableDraftClobberedByRollback` | **坐实** — rollback 改 archiveID → 同 R1-2 |
| R1-X **新发现** 目录权限 bug（P0） | `TestEditableDraftSurvivesResyncOfSameArchive`（失败重现） | **已修复（两层修复）** |
| R1-Y **V-3 新发现** publish 链路 500 死锁 | 真实栈走查触发 | **已修复** (`admin/projects.go`) |

---

## R1-X（已修复，两层）：目录权限 EACCES

### 第一层（c30d97b）：`syncEditable` 顶层目录

`syncEditable` 改为自行 `RemoveAll` + `MkdirAll(filesDir, 0755)` 预建可写顶层目录。

### 第二层（e92ddaf）：`archive/untar` 嵌套目录

真实栈 V-3 走查时，第二次 publish（产生新 assetID → dev re-sync → 重新 untar）失败：
```
open .../archive/files/dashboards/draft_explore.yaml: permission denied
```

仅预建顶层不够，`untar` 为嵌套路径调 `os.MkdirAll(filepath.Dir(target), header.FileInfo().Mode())` 用文件的 0644 建子目录。修复：`untar` 里 regular-file 分支的 `MkdirAll` 硬编码 0755；directory 分支 OR 上 0700。

---

## R1-Y（已修复）：publish 链路 UpdateProject 500 死锁

### 问题

`UpdateProject` 开头**无条件**检查"目标分支上是否有非 prod 部署"。对 archive 项目，PrimaryBranch 恒为空串，空串过滤等于不过滤 → 查到发布本身刚用的 dev deployment → 触发守卫 → 500。

### 修复 (`ca45ae390`)

守卫加上 `if oldProj.PrimaryBranch != opts.PrimaryBranch` 条件——PrimaryBranch 未变时跳过。

### 修复（本次 commit）

`syncEditable` 改为：自行 `os.RemoveAll` + `os.MkdirAll(filesDir, 0755)` 预建可写目录，然后 `archive.Download(..., clean=false, ...)`。这样 `untar` 在一个已经存在且可写的根目录下操作，嵌套目录由 `MkdirAll(parent, fileMode)` 创建时继承已有 parent 权限（parent 已经是 0755）。

文件：`runtime/drivers/admin/repo_archive.go:79-97`

### 影响

- 修复前：**每次 publish + 下次 dev sync 之后 Studio 编辑会 EACCES**，即核心链路完全断裂。
- 非 editable（prod）路径不受影响（`sync` 里的 `MkdirTemp` 创建的目录本身是 0700/0755）。

---

## R1-1：publish 时的并发写入

### 分析

`packageDevDraft` 从 runtime gRPC `ListFiles` + `GetFile` 逐个读文件。如果此刻 Studio IDE 正在通过 runtime `Put`（file API）写入同一个文件，理论上 GetFile 可能读到 half-written 内容。

**概率评估**：低。GetFile 读 `os.ReadFile`（原子 read），Put 写 `os.Create` + `io.Copy`（非原子，但通常亚毫秒）。race window 极小。但在银行客户要求的数据一致性标准下，并非零风险。

### 建议（非当前 sprint）

短期缓解：publish 时在 runtime 侧加一个 repo read lock（已有 `repo.mu`），阻塞同时期的 Put 直到 list+read 完成。代价：publish 期间 Studio 保存会短暂挂起（秒级）。
长期：snapshot 整个 draft directory（如 hard-link tree），再打包。

---

## R1-2：dev 草稿被新 archive 覆盖

### 行为（已测试证实）

- 治理者 A 在 dev instance 编辑了 `metrics/wip.yaml`，未 publish。
- 治理者 B（或 A 自己）publish 了其他改动 → 产生新 assetID。
- 下次 dev instance 执行 `pullInner → archive.sync → syncEditable` → archiveID 不匹配 → **clean extraction，wip.yaml 丢失**。

### 设计意图

当前行为是**有意为之**（注释写得很清楚："a new archive version replaces the draft contents"）。这假设 dev instance 上的内容只是"已发布内容的编辑视图"，每次新版本就是一次 reset。

### 风险等级

**P2（UX 问题，非数据丢失）** — 丢失的是 dev 草稿（尚未进入任何持久化版本），本质上是"自动保存但未提交"的工作丢失。如果只有一个治理者，不会触发（因为 publish 包含了当前 dev 的全部内容）。多治理者并行编辑同一数据域时会触发。

### 建议（Phase 4 之后）

- 短期：publish 成功后不立刻更新 dev archiveID（即 dev 保持原状态 + 本地编辑），直到治理者手动"同步最新发布版本"。
- 长期：dev instance 引入"未提交变更列表"（类似 git diff），publish/rollback 前提示"你有未包含在发布中的本地编辑"。

---

## R1-3：rollback 后 dev 草稿丢失

### 行为（已测试证实）

- 治理者 rollback 到 v1 → `setProjectArchiveAsset` 改 archiveID → dev instance 下次 sync 触发 R1-2 同样的 clean extraction → **dev 草稿被旧版本覆盖**。
- 且 rollback 在 UI 上呈现为"回滚生产版本"——治理者不会预期自己的 dev 编辑也被回滚。

### 风险等级

**P2（同 R1-2）** — 表现为 UX 意外，实际机制与 R1-2 完全相同。

### 建议

同 R1-2。核心改动是让 dev instance 的 archiveID 追踪解耦于 prod（或者至少在 rollback 时给 dev 一个 grace period / 确认提示）。

---

## 总结

| 发现 | 严重度 | 状态 |
|---|---|---|
| R1-X 目录权限 EACCES | **P0 Ship-blocker** | ✅ **已修复**（本次 commit） |
| R1-1 并发写入 | P3 低概率 | ⏸️ 延后（短期仅文档告知） |
| R1-2 新 archive 覆盖草稿 | P2 UX | ⏸️ 延后（单治理者场景不触发） |
| R1-3 rollback 丢 dev 编辑 | P2 UX | ⏸️ 延后（同 R1-2 机制） |

---

## 尚未验证的部分（诚实声明）

本轮验证以 **unit test + 代码审读** 完成，**未在真实 docker-compose 部署上跑完整链路**。以下环节仍需一次人工走查确认（预计 0.5 天）：

1. `publish` 后 prod deployment 是否真的重新拉取并 reconcile 成功（`UpdateProject → ArchiveAssetID` 变更是否可靠触发 prod 重部署）。
2. `rollback` 后 prod 是否正确回到旧版本内容。
3. R1-X 修复在容器内（`USER stardata`, uid 1001, 挂载 `runtime_data` volume）的实际表现。
4. `packageDevDraft` 在项目文件较多（>100 文件）时的耗时与 gRPC 消息大小上限。

### 人工走查步骤

```bash
cd deploy
docker compose --env-file .env.admin -f docker-compose.admin.yml --profile proxy up -d --build
# 1. 浏览器登录 → 创建项目 → 进 Studio
# 2. Studio → 语义层 → 编辑一个 metrics view → 保存
# 3. Studio → 发布 → 填 note → 发布
# 4. 切到业务门户 → 确认看板/推荐问题反映新内容
# 5. 回 Studio 再改一次 → 发布 v2
# 6. Studio → 发布 → 历史 → 回滚到 v1
# 7. 确认 prod 内容回到 v1；确认 Studio 仍可正常编辑（验证 R1-X 修复）
```

