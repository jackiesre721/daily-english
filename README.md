# Daily English

一款英语精读学习工具，通过阅读真实语料积累词汇、提升英语能力。

## 功能概览

### 阅读学习
- **单词即点即查**：点击文章中任意单词，左侧面板显示音标、中文释义、英文解释
- **句子 AI 翻译**：选中句子可一键获取 AI 中文翻译，翻译结果自动保存到文章
- **AI 文本解析**：选中任意文本进行 AI 深度分析，自动提取生词和短语，保存到生词本

### 词汇管理
- **生词本**：收录所有学习过程中遇到的单词，支持搜索和浏览
- **多来源查词**：优先使用本地 ECDICT 词典（340 万词条），未收录时回退到 AI 查询
- **学习追踪**：单词与文章关联，记录学习来源和上下文
- **短语收录**：AI 解析自动提取短语，记录出处和例句

### 文章导入
- **文本导入**：粘贴英文原文，AI 自动生成分段、翻译、词汇解析
- **字幕导入**：支持 SRT 字幕文件导入，自动识别对话角色
- **网页导入**：通过 URL 抓取网页内容
- **来源管理**：按导入来源筛选和浏览文章

### 个性化设置
- **5 套主题配色**：莫兰迪暖调、自然大地色、静谧蓝调、护眼经典、默认绿，即选即用
- **AI 接口配置**：支持自定义 AI 服务端点（兼容 OpenAI API 格式）

## 安装

### macOS 桌面应用（推荐）

从 [Releases](../../releases) 下载最新 DMG，双击打开后将 `Daily English.app` 拖入 Applications。

或使用 Homebrew：

```bash
brew tap richie/tap
brew install --cask daily-english
```

### Web 模式

需要 Go 1.21+。

```bash
git clone https://github.com/richie/daily-english.git
cd daily-english
./start.sh
```

浏览器访问 `http://localhost:8080`。

## 开发

### 环境要求

- Go 1.21+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)（桌面应用开发）

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 目录结构

```
├── main.go                  # Wails 桌面应用入口
├── cmd/server/main.go       # Web 服务器入口
├── wails.json               # Wails 项目配置
├── internal/
│   ├── ai/client.go         # AI 客户端（OpenAI / Anthropic）
│   ├── config/config.go     # 配置加载
│   ├── handler/             # 12 个 HTTP handler
│   ├── model/               # 数据模型
│   ├── repository/          # 数据访问层（SQLite）
│   └── router/router.go     # 路由注册
├── static/                  # 前端 HTML/CSS/JS
├── migrations/              # SQLite 迁移脚本
├── build/                   # 构建产物（图标、DMG、.app）
├── homebrew-tap/            # Homebrew Cask 公式
└── start.sh                 # Web 模式启动脚本
```

### 运行开发模式

桌面应用（热重载）：

```bash
wails dev
```

Web 服务器：

```bash
./start.sh          # 构建并启动
./start.sh stop     # 停止
./start.sh restart  # 重启
```

### 构建桌面应用

```bash
wails build                        # 当前架构
wails build -platform darwin/arm64 # Apple Silicon
wails build -platform darwin/universal  # 通用二进制
```

产物在 `build/bin/Daily English.app`。

### 构建可分发 DMG

```bash
# 本地构建 DMG
VERSION="1.0.0"
DMG_DIR=$(mktemp -d)
cp -R "build/bin/Daily English.app" "${DMG_DIR}/"
ln -sf /Applications "${DMG_DIR}/Applications"
hdiutil create -volname "Daily English" -srcfolder "${DMG_DIR}" -ov -format UDZO "build/Daily-English-${VERSION}.dmg"
```

或通过 CI 自动构建：推送 tag 触发 GitHub Actions。

```bash
git tag v1.0.0
git push origin v1.0.0
```

## 配置

### AI 接口

在应用的「设置」页面配置：

| 字段 | 说明 |
|------|------|
| API Base URL | AI 服务地址（如 `https://api.openai.com`） |
| Auth Token | API 密钥 |
| Model | 模型名称（如 `gpt-4o`） |

支持 OpenAI 和 Anthropic API 格式，自动检测。

### 离线词典（可选）

下载 [ECDICT](https://github.com/skywind3000/ECDICT) 词典数据库，放置到：

- 桌面应用：`~/Library/Application Support/DailyEnglish/ecdict.db`
- Web 模式：`./data/ecdict.db`

### 数据存储

| 模式 | 位置 |
|------|------|
| 桌面应用 | `~/Library/Application Support/DailyEnglish/dailyenglish.db` |
| Web 模式 | `./data/dailyenglish.db` |

支持在设置页面导出/导入数据备份。

## 技术栈

- **后端**：Go + Gin + SQLite（modernc.org/sqlite，纯 Go 驱动）
- **前端**：纯 HTML/CSS/JS（无框架依赖）
- **桌面**：[Wails v2](https://wails.io)（Go + WebView）
- **词典**：ECDICT 离线词典（340 万词条）
- **AI**：OpenAI / Anthropic API

## License

MIT
