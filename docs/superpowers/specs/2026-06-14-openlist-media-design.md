# OpenList Media 设计

## 背景

当前家庭影院由 MoviePilot、OpenList 和 Jellyfin 组成：

- MoviePilot 提供订阅监控、资源识别、下载触发和刮削。
- OpenList 提供文件管理、存储抽象、离线下载适配器和跨存储移动能力。
- Jellyfin 主要服务 Apple TV 客户端播放，不应承担编排职责。

真正需要保留的是 MoviePilot 的订阅与刮削体验，尤其是刮削；不需要 MoviePilot 的站点认证、站点索引和插件市场复杂度。更合适的方向是在 OpenList fork 中加入一个媒体自动化模块，让 OpenList 成为文件管理和媒体入库的统一入口。

## 目标

第一版目标是构建 `OpenList Media`：

- 监听自定义 RSS/Atom feed。
- 按订阅规则匹配 feed item。
- 通过 OpenList 已有下载适配器创建下载任务。
- 下载完成后整理到规范媒体库目录。
- 使用 TMDB 识别并刮削元数据。
- 生成 Jellyfin、Emby、Plex 可消费的本地元数据。
- 通过可插拔媒体库适配器通知外部媒体库刷新。

第一版只支持 TMDB。豆瓣、Bangumi、多站点搜索和私有 tracker 站点能力不进入第一版。

## 非目标

- 不绕过 MoviePilot 或第三方站点认证。
- 不复刻 MoviePilot 的站点系统。
- 不把 Jellyfin 作为核心依赖。
- 不深度绑定某一个媒体库服务。
- 不在第一版实现豆瓣、Bangumi、洗版、站点搜索或插件市场。
- 不要求 Jellyfin 通过 OpenList WebDAV 读取文件；推荐媒体库直接读取最终入库路径。

## 推荐方案

采用 OpenList fork 内置媒体模块：

```text
Feed Source
  -> Release Parser
  -> Subscription Matcher
  -> OpenList Downloader
  -> Download Task
  -> Organizer
  -> TMDB Scraper
  -> Media Server Adapter
```

媒体模块只依赖 OpenList 的稳定能力：

- `internal/fs`：移动、复制、重命名、目录创建。
- `internal/offline_download/tool`：下载适配器入口。
- `pkg/task` 与 `internal/task`：任务状态、重试和持久化。
- GORM 数据层：配置、订阅、release、任务、刮削记录。
- OpenList Frontend：新增一级媒体管理入口。

## 设计原则

1. 媒体领域与下载器实现解耦。媒体模块只发出下载请求，不关心底层是 qBittorrent、Transmission 还是 aria2。
2. 媒体库服务是外部消费者。OpenList 负责整理和元数据，媒体库只负责扫描与播放。
3. 刮削体验优先。订阅第一版可以轻量，刮削要作为第一版成败标准。
4. 所有自动行为都要可追踪。release 命中、下载任务、整理任务、刮削任务和刷新结果都要有记录。
5. 失败进入待处理列表。识别失败、多候选、下载失败和刮削失败不能静默丢失。

## 低侵入集成策略

OpenList 是这个 fork 的核心，媒体能力应作为独立功能域挂载到 OpenList，而不是把订阅、整理和刮削逻辑散进既有文件管理、下载器或任务模块。这样后续同步上游时，冲突范围会更小，也更容易判断哪些变更属于 media fork。

后端约束：

- 媒体领域代码默认放在 `internal/media/...`，包括 feed、subscription、download、organizer、recognizer、scraper、server adapter、service 和 repository。
- 访问 OpenList 既有能力时通过薄 adapter 进入，例如 `internal/media/download` 只依赖一个注入的 add-url 函数，具体桥接到 `internal/offline_download/tool` 的代码集中在单个适配文件。
- 避免修改 OpenList 既有下载器、存储、文件系统和任务语义。确实需要接入时，只做注册、回调或组合根级别的薄桥接。
- 后续不可避免的核心接入点应限制为少量文件，例如路由注册、数据库迁移注册、任务调度启动、文件列表操作菜单注册。每个实施计划都必须单列这些接入点。
- 如果现有模块没有合适扩展点，优先新增小型通用注册口，再让 media 模块挂载；不要在多个既有 handler 或 service 中直接写 media 分支逻辑。

前端约束：

- Media 页面默认集中在独立目录，例如 `src/pages/manage/media/...` 或当前前端仓库最接近的管理页目录。
- 既有前端只允许少量导航、路由和 i18n 入口修改，例如侧边栏菜单、route registry、语言文件。
- 文件列表右键入口只做命令注册和跳转，刮削、整理和确认流程放回 Media 页面或独立 media 组件内。

每个后续里程碑都要维护一个“OpenList 原模块触点”清单。若某个任务需要同时修改多个既有核心文件，应先重新评估是否能通过 adapter、registry 或独立 service 缩小侵入面。

## 模块划分

### `internal/media/feed`

负责 feed 配置、抓取和解析。

能力：

- 支持 RSS 和 Atom。
- 解析 `title`、`description`、`link`、`enclosure`、`pubDate`、`size`。
- 支持代理、刷新周期、启用状态。
- 生成稳定 fingerprint 用于 release 去重。

### `internal/media/subscription`

负责订阅规则和匹配。

能力：

- 关键词匹配。
- include/exclude 正则。
- 文件大小范围。
- 电影和剧集订阅。
- 每条订阅可覆盖默认下载器、下载路径、媒体库路径和整理模式。
- 命中后创建 release 记录。

第一版参考 MoviePilot `自定义订阅` 插件行为，但作为 OpenList 一级功能实现。

### `internal/media/download`

负责把 release 转换为 OpenList 下载任务。

媒体模块的下载抽象保持很薄：

```go
type Downloader interface {
	Add(ctx context.Context, req DownloadRequest) (*DownloadTaskRef, error)
}

type DownloadRequest struct {
	URL           string
	DownloadPath  string
	DownloaderKey string
	SubscriptionID uint
	ReleaseID      uint
}
```

OpenList 适配实现复用已有入口：

```go
tool.AddURL(ctx, &tool.AddURLArgs{
	URL:        req.URL,
	DstDirPath: req.DownloadPath,
	Tool:       req.DownloaderKey,
})
```

`DownloaderKey` 指用户在 OpenList 中配置的下载器名称。媒体模块不把 qBittorrent 作为领域模型。

实际桥接到 OpenList 下载器时应集中在一个组合根或 adapter 文件中，避免 media 以外的业务模块直接感知 subscription、release 或 scrape 状态。

### `internal/media/organizer`

负责下载完成后的文件发现、识别、重命名和入库。

推荐两段式路径：

```text
/downloads/incoming
  -> download output
/media/Movies
  -> organized movie library
/media/Shows
  -> organized show library
```

整理模式：

- `move`
- `copy`
- `hardlink`

第一版必须支持 `move` 和 `copy`。`hardlink` 需要确认源和目标是否在同一文件系统，不能作为默认策略。

### `internal/media/recognizer`

负责从文件名、目录名和 feed 标题中识别媒体信息。

第一版能力：

- 电影名、年份、版本、分辨率。
- 剧集名、季号、集号、集范围。
- 手动指定 `tmdbid` 作为兜底。
- 自动识别失败或多候选时进入待确认状态。
- 用户确认后缓存 title/path 到 TMDB 的映射。

### `internal/media/scraper/tmdb`

负责 TMDB 查询、图片下载和本地 metadata 输出。

输出目标是媒体库通用格式：

```text
Movies/Movie Name (2024)/Movie Name (2024).mkv
Movies/Movie Name (2024)/movie.nfo
Movies/Movie Name (2024)/poster.jpg
Movies/Movie Name (2024)/fanart.jpg

Shows/Show Name (2024)/tvshow.nfo
Shows/Show Name (2024)/poster.jpg
Shows/Show Name (2024)/fanart.jpg
Shows/Show Name (2024)/Season 01/Show Name - S01E01.mkv
Shows/Show Name (2024)/Season 01/Show Name - S01E01.nfo
Shows/Show Name (2024)/Season 01/season01-poster.jpg
```

刮削写入策略：

- 默认不覆盖用户手工修改过的 `nfo`。
- 图片缺失时补齐。
- 用户可在手动刮削入口选择强制覆盖。
- TMDB API 失败时保留整理结果，并把刮削任务标记为可重试。

### `internal/media/server`

负责媒体库服务适配。

接口保持小：

```go
type MediaServer interface {
	Name() string
	Test(ctx context.Context) error
	RefreshLibrary(ctx context.Context, libraryID string) error
	RefreshPath(ctx context.Context, path string) error
	QueryItem(ctx context.Context, identity MediaIdentity) (*LibraryItem, error)
}
```

第一版只实现 Jellyfin。数据库和 API 按多媒体库适配器设计，后续可接 Emby 或 Plex。

## 数据模型

新增表建议：

| 表 | 作用 |
| --- | --- |
| `media_feed_sources` | RSS/Atom 源配置 |
| `media_subscriptions` | 订阅规则 |
| `media_releases` | feed item 去重、匹配和下载状态 |
| `media_download_refs` | release 到 OpenList 下载任务的映射 |
| `media_organize_jobs` | 整理任务状态和结果 |
| `media_scrape_jobs` | TMDB 刮削任务状态和错误 |
| `media_identity_mappings` | 用户确认后的识别缓存 |
| `media_servers` | Jellyfin/Emby/Plex 配置 |
| `media_server_libraries` | 外部媒体库配置 |
| `media_library_mappings` | OpenList 路径到媒体库路径的映射 |

核心状态流：

```text
release.pending
  -> release.matched
  -> download.created
  -> download.completed
  -> organize.completed
  -> scrape.completed
  -> refresh.completed
```

失败状态：

```text
release.ignored
release.duplicate
recognize.pending_confirmation
download.failed
organize.failed
scrape.failed
refresh.failed
```

## API 设计

后端新增 `/api/admin/media` 分组。

第一版 API：

- `GET /api/admin/media/feeds`
- `POST /api/admin/media/feeds`
- `POST /api/admin/media/feeds/:id/refresh`
- `GET /api/admin/media/subscriptions`
- `POST /api/admin/media/subscriptions`
- `POST /api/admin/media/subscriptions/:id/test`
- `GET /api/admin/media/releases`
- `POST /api/admin/media/releases/:id/download`
- `GET /api/admin/media/jobs`
- `POST /api/admin/media/jobs/:id/retry`
- `POST /api/admin/media/scrape/path`
- `POST /api/admin/media/scrape/confirm`
- `GET /api/admin/media/servers`
- `POST /api/admin/media/servers`
- `POST /api/admin/media/servers/:id/test`
- `POST /api/admin/media/servers/:id/refresh`

手动刮削要支持文件或目录路径输入，适合从 OpenList 文件列表右键菜单进入。

## 前端设计

OpenList Frontend 新增一级管理菜单 `Media`。

页面：

- `Dashboard`：订阅命中、下载、整理、刮削和刷新概览。
- `Subscriptions`：订阅列表、规则、默认路径和下载器。
- `Feed Sources`：RSS/Atom 源配置和手动刷新。
- `Releases`：已解析 item、命中结果、下载状态、去重原因。
- `Pending Confirmation`：识别失败和多候选确认。
- `Scrape`：手动路径刮削、覆盖策略、结果预览。
- `Media Servers`：Jellyfin 连接、媒体库映射和刷新测试。
- `Settings`：TMDB API key、默认下载路径、默认媒体库路径、整理模式。

文件列表增加两个入口：

- `Scrape metadata`
- `Organize to media library`

## 配置

第一版全局配置：

- TMDB API key。
- 默认下载器。
- 默认下载路径。
- 默认电影媒体库路径。
- 默认剧集媒体库路径。
- 默认整理模式。
- 默认 feed 刷新间隔。
- 默认图片语言。
- 默认元数据语言。
- Jellyfin 地址和 API key。

语言默认：

- 元数据语言：`zh-CN`。
- 图片 fallback：`zh-CN` -> `en-US` -> any。

## 任务与调度

需要新增媒体任务管理器：

- feed refresh task
- subscription match task
- organize task
- scrape task
- media server refresh task

任务应复用 OpenList 的任务模型，支持：

- 当前任务列表。
- 历史任务列表。
- 取消。
- 重试。
- 错误信息。
- 进度。

feed refresh 可以用定时调度触发，也可以由用户手动触发。

## 错误处理

失败不应中断后续可执行步骤：

- 下载成功但整理失败：保留下载任务引用，允许重试整理。
- 整理成功但刮削失败：保留文件，允许重试刮削。
- 刮削成功但媒体库刷新失败：保留 metadata，允许重试刷新。
- TMDB 多候选：进入待确认，不自动选择低置信度结果。
- feed 解析失败：记录错误，不删除历史 release。

## 验证计划

后端验证：

- feed parser 单元测试：RSS、Atom、缺少 enclosure、重复 item。
- subscription matcher 单元测试：include、exclude、大小范围、剧集匹配。
- recognizer 单元测试：电影、单集、多集、整季、异常命名。
- TMDB scraper 单元测试：API client mock、NFO 输出、图片 fallback。
- organizer 单元测试：电影路径、剧集路径、冲突处理、覆盖策略。
- media server adapter 单元测试：Jellyfin test、refresh library、refresh path。

集成验证：

- 使用测试 RSS 创建 release。
- 使用 SimpleHttp 或 qBittorrent 创建下载任务。
- 模拟下载完成后整理到测试媒体库路径。
- 对测试文件执行 TMDB 刮削。
- 验证 NFO 和图片文件存在。
- 验证 Jellyfin refresh API 被调用。

前端验证：

- 订阅增删改查。
- feed 手动刷新。
- release 状态筛选。
- 待确认选择候选。
- 手动刮削路径。
- 媒体库连接测试。

## 实施顺序

1. 建立低侵入边界：每个里程碑先声明 `internal/media` 新增内容和 OpenList 原模块触点。
2. feed source 和 release parser。
3. subscription matcher。
4. OpenList downloader adapter。
5. 后端数据模型和迁移注册。
6. 下载任务完成监听和 release 状态回写。
7. organizer。
8. TMDB client 和 scraper。
9. Jellyfin adapter。
10. media API。
11. 前端 Media 菜单和页面。
12. 文件列表右键入口。
13. 集成测试和本地部署验证。

## 风险

最大风险是下载完成事件与整理触发的边界。OpenList 现有下载任务已经有 transfer 阶段，但媒体模块需要知道哪个 release 对应哪个完成目录。第一版应通过 `media_download_refs` 明确记录任务 ID、下载路径和 release ID，避免靠路径猜测。

第二个风险是刮削体验低于 MoviePilot。第一版必须把识别失败、多候选确认、手动指定 TMDB ID 和可重试任务做完整，否则用户会频繁回到 MoviePilot。

第三个风险是前端仓库独立。OpenList 后端构建时默认拉取 `OpenListTeam/OpenList-Frontend` release。fork 后需要配置 `FRONTEND_REPO` 指向自己的前端 fork，或在构建流程中使用本地前端 dist。

第四个风险是对 OpenList 核心侵入过深，导致同步上游更新时频繁冲突。缓解方式是把媒体领域逻辑稳定放在 `internal/media`，把核心修改限制为注册和 adapter，并在每个计划中记录不可避免的核心触点。

## 决策

- 采用 OpenList fork 内置媒体模块。
- 以低侵入方式集成：媒体领域逻辑集中在 `internal/media`，OpenList 既有模块只保留少量注册和 adapter 触点。
- 第一版只支持 TMDB。
- 订阅参考 MoviePilot `自定义订阅` 插件行为。
- 下载复用 OpenList 已有下载适配器，不把 qBittorrent 写入媒体领域模型。
- 媒体库服务通过 adapter 抽象，第一版实现 Jellyfin。
- Jellyfin 不作为核心依赖，只作为可配置媒体库服务。
- 实现范围同时包含 OpenList 后端和 OpenList Frontend。
