# VelaUX 本地开发指南：CI 流水线模板修复与本地运行

## 背景

在 VelaUX 中新增一个 CI 流水线模板（代码扫描、编译打包、推送镜像），原始模板由 AI 生成但无法工作。本文档记录了问题定位、修复过程，以及如何在本地运行 VelaUX 进行功能验证。

---

## 一、CI 流水线模板修复

### 1.1 问题定位

原始模板文件：`packages/velaux-ui/src/pages/PipelineListPage/components/CreatePipeline/pipeline-template.ts`

原始模板存在 4 个致命问题：

| 问题 | 说明 |
|------|------|
| `type: build-pipeline` 不存在 | KubeVela 没有名为 `build-pipeline` 的 WorkflowStepDefinition，这是 AI 凭空编造的步骤类型 |
| `properties` 结构虚构 | `mainContainer`、`cwd`、`args`（字符串）、`resources`、`env` 等字段不属于任何真实步骤类型 |
| `${context.xxx}` 在 properties 中不生效 | Pipeline context 值只能在 `if` 条件中作为 CUE 表达式使用，不能在 properties 中做字符串插值 |
| `inputs`/`outputs` 引用无效 | `output.codeInfo`、`output.image` 等不对应任何真实步骤的输出 |

### 1.2 KubeVela 真实的工作流步骤类型

步骤类型定义位置：`kubevela/vela-templates/definitions/internal/workflowstep/`

与 CI 相关的真实步骤类型：

| 步骤类型 | 用途 | 参数 |
|---------|------|------|
| `vela-cli` | 创建 K8s Job 运行任意容器命令 | `image`, `command`(数组), `serviceAccountName`, `storage` |
| `build-push-image` | 用 Kaniko 从 Git 仓库构建并推送 Docker 镜像 | `image`, `context`({git,branch}), `dockerfile`, `credentials` |
| `clean-jobs` | 清理流水线产生的 Job 和 Pod | `namespace`, `labelselector` |
| `print-message-in-status` | 在步骤状态中输出消息 | `message` |
| `step-group` | 并行执行子步骤 | `subSteps` |
| `notification` | 发送通知（Slack/钉钉/飞书/邮件） | `slack`, `dingding`, `lark`, `email` |

### 1.3 修复后的模板

```yaml
- name: Code Scan
  type: vela-cli
  properties:
    image: aquasec/trivy:0.50.0
    command:
      - sh
      - -c
      - "trivy repo --exit-code 1 --severity HIGH,CRITICAL https://github.com/myorg/myapp.git"
    serviceAccountName: default

- name: Build and Push Image
  type: build-push-image
  properties:
    image: docker.io/myorg/myapp:latest
    context:
      git: https://github.com/myorg/myapp.git
      branch: main
    dockerfile: ./Dockerfile
    credentials:
      image:
        name: docker-registry-secret
        key: .dockerconfigjson

- name: Clean
  type: clean-jobs

- name: Notification
  type: print-message-in-status
  properties:
    message: "CI completed: docker.io/myorg/myapp:latest built and pushed."
```

关键区别：
- **Code Scan**：`vela-cli` 类型创建 K8s Job，用 Trivy 直接扫描远程 Git 仓库，发现 HIGH/CRITICAL 漏洞时 exit code 1 中断流水线
- **Build and Push Image**：`build-push-image` 类型内部使用 Kaniko Pod，从 Git 上下文构建镜像并推送，通过 K8s Secret 做镜像仓库认证
- **Clean**：`clean-jobs` 清理流水线执行过程中创建的 Job/Pod 资源

### 1.4 使用前准备

创建流水线后，在 Pipeline Studio 中将占位值改为实际值：

1. Git 仓库地址（Code Scan 和 Build and Push Image 两个步骤中都要改）
2. 目标镜像地址（`docker.io/myorg/myapp:latest`）
3. 提前创建镜像仓库凭证 Secret：
   ```bash
   kubectl create secret docker-registry docker-registry-secret \
     --docker-server=docker.io \
     --docker-username=<username> \
     --docker-password=<password> \
     -n <pipeline-namespace>
   ```

---

## 二、本地运行 VelaUX

### 2.1 前提条件

- 本地 K8s 集群已安装 vela-core 和 vela-workflow
- Go 1.19+、Node.js 18/20、Yarn 3+

### 2.2 缩容集群中的 velaux-server

本地和集群中的 velaux-server 会竞争 leader 选举锁，必须先缩容：

```bash
kubectl scale deploy velaux-server -n vela-system --replicas=0
```

### 2.3 解决 Node.js 兼容性问题

项目使用 Yarn 3 PnP 模式，与 Node.js 22 存在兼容性问题。解决方案有两个：

**方案 A（推荐）：切换 nodeLinker 为 node-modules**

编辑 `.yarnrc.yml`：
```yaml
nodeLinker: node-modules     # 原来是 pnp
yarnPath: .yarn/releases/yarn-3.8.7.cjs
```

需要同步修改 webpack 配置，让 esbuild-loader 处理 `@velaux/` workspace 包：

`scripts/webpack/webpack.common.js` 和 `scripts/webpack/webpack.dev.js` 中：
```js
// 修改前
exclude: /node_modules/,

// 修改后
exclude: /node_modules\/(?!@velaux)/,
```

原因：PnP 模式下 workspace 包通过虚拟路径解析，不经过 `node_modules/`。切换到 node-modules 模式后，workspace 包被放入 `node_modules/@velaux/`，需要排除在 exclude 规则之外才能被 esbuild-loader 编译。

**方案 B：保持 PnP，使用 Node.js 18**

如果不想改 nodeLinker，用 Node 18 可以避开大部分兼容性问题。

### 2.4 升级 Yarn（如原版本 < 3.8）

```bash
cd velaux
corepack enable
yarn set version 3.8.7
```

Yarn 3.6.0 在 Node.js 22 上会报 `fastqueue concurrency must be greater than 1` 错误。

### 2.5 安装依赖 & 构建子包

```bash
cd velaux
yarn install
```

子包（`@velaux/theme`、`@velaux/data`、`@velaux/ui`）需要先构建，否则主 UI 引用不到类型定义：

```bash
# velaux-theme 使用 webpack 4，需要 Node 18 + legacy OpenSSL
nvm use 18
export NODE_OPTIONS=--openssl-legacy-provider
yarn build-packages
```

### 2.6 启动前端（终端 1）

```bash
nvm use 20
unset NODE_OPTIONS
cd velaux
yarn dev
```

webpack 以 watch 模式运行，修改源码后自动重新编译到 `public/build/`。看到 `compiled successfully` 即表示构建成功。

### 2.7 启动后端（终端 2）

```bash
cd velaux
make run-server
# 等价于: go run ./cmd/server/main.go
```

后端通过本地 kubeconfig 连接 K8s 集群，默认监听 `0.0.0.0:8000`，同时提供 API 和前端静态文件。

### 2.8 访问

浏览器打开 **http://localhost:8000**

架构示意：
```
浏览器 (localhost:8000)
    │
    ├── /api/v1/*          →  Go 后端 (REST API)  →  K8s API (via kubeconfig)
    ├── /public/build/*    →  Go 后端 (静态文件 from ./public/build/)
    └── /*                 →  Go 后端 (rewrite → index.html, SPA 路由)
```

前端 `APIBASE` 默认为空字符串（`process.env.BASE_DOMAIN || ''`），所有 API 请求发到同源地址，无需配置 proxy。

### 2.9 验证模板改动

1. 进入 Pipeline 页面，点击 "New Pipeline"
2. 在 Template 下拉框中选择 **"CI Template: Scan, Build & Push Image"**
3. 填写 Name、Project，提交
4. 创建成功后自动跳转到 Pipeline Studio，可查看步骤拓扑并编辑

### 2.10 恢复集群

验证完成后恢复集群中的 velaux：

```bash
kubectl scale deploy velaux-server -n vela-system --replicas=1
```

---

## 三、关键文件索引

| 文件 | 作用 |
|------|------|
| `packages/velaux-ui/src/pages/PipelineListPage/components/CreatePipeline/pipeline-template.ts` | 流水线模板定义（前端） |
| `packages/velaux-ui/src/pages/PipelineListPage/components/CreatePipeline/index.tsx` | 创建流水线抽屉组件 |
| `packages/velaux-ui/src/api/pipeline.ts` | 流水线 API 客户端 |
| `pkg/server/interfaces/api/pipeline.go` | 流水线 REST API handler（后端） |
| `pkg/server/domain/service/pipeline.go` | 流水线业务逻辑 |
| `pkg/server/domain/model/pipeline.go` | 流水线数据模型 |
| `kubevela/vela-templates/definitions/internal/workflowstep/` | 工作流步骤类型 CUE 定义 |
| `scripts/webpack/webpack.common.js` | webpack 公共配置 |
| `scripts/webpack/webpack.dev.js` | webpack 开发配置 |
| `.yarnrc.yml` | Yarn Berry 配置 |
