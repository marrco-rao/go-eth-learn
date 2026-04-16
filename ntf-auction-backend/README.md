# ntf-auction-backend

一个基于 Gin + GORM + JWT 的简化后端服务。

## 功能特性

- Gin HTTP 服务
- YAML 配置文件
- 数据库支持：
  - SQLite（默认）
  - MySQL
- 统一响应结构与错误码
- 基于 JWT 的认证机制
- 请求日志落库到 `logs` 表
- 提供登录接口与日志分页查询接口

## 项目结构

```text
ntf-auction-backend/
  cmd/
    main.go
  config/
    app.yaml
  data/
    app.db（SQLite 运行时自动生成）
  internal/
    app/
    apperr/
    config/
    database/
    handler/http/
    logger/
    middleware/
    model/
    repository/
    response/
    service/
  testdata/sql/
    user_tables.sql
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 使用默认 SQLite 启动

```bash
go run ./cmd
```

服务默认地址：`http://127.0.0.1:8080`

### 3. 健康检查

```bash
curl -s http://127.0.0.1:8080/healthz
```

## 配置说明

配置文件：`config/app.yaml`

```yaml
server:
  port: 8080

database:
  driver: sqlite
  sqlite_path: data/app.db
  mysql_dsn: root:password@tcp(127.0.0.1:3306)/ntf_auction?charset=utf8mb4&parseTime=True&loc=Local

jwt:
  secret: change-this-secret
  expire_hours: 24

default_user:
  username: admin
  password: admin123

ethereum:
  # HTTP RPC 地址（应用启动必需）
  rpc_url: http://127.0.0.1:8545
  # 仅发送交易接口需要；发送地址会由私钥自动推导
  tx_signer_private_key: ""
```

也可以使用环境变量覆盖：

```bash
export ETH_RPC_URL="https://sepolia.infura.io/v3/<your-key>"
export ETH_TX_SIGNER_PRIVATE_KEY="<your-private-key-hex>"
```

说明：`ETH_TX_SIGNER_PRIVATE_KEY` 仅在调用发送交易接口时必需；查询类接口不依赖该配置。

## 切换到 MySQL

1. 先创建数据库，例如 `ntf_auction`。
2. 修改 `config/app.yaml`：

```yaml
database:
  driver: mysql
  mysql_dsn: your_user:your_password@tcp(127.0.0.1:3306)/ntf_auction?charset=utf8mb4&parseTime=True&loc=Local
```

3. 启动服务：

```bash
go run ./cmd
```

## 链上拍卖事件订阅（go-ethereum）

建议把链上监听放在 `internal/service/`，因为它属于业务流程的一部分：订阅链上事件并转换成应用可消费的数据（本例中写入 `logs` 表）。

本项目示例实现：

- `internal/service/auction_event_subscriber.go`
- 启动接入点在 `internal/app/app.go` 的 `Run()` 中自动后台启动

### 环境变量

当下面两个环境变量都存在时，服务启动后会自动订阅 `NFTAuctionListed` 事件：

```bash
export ETH_WS_URL="wss://sepolia.infura.io/ws/v3/<your-key>"
export MARKETPLACE_CONTRACT_ADDRESS="0x1234567890abcdef1234567890abcdef12345678"
```

然后正常启动：

```bash
go run ./cmd
```

未配置环境变量时，订阅器默认关闭，不影响 HTTP 服务启动。

## 接口示例

基础路径：`/api/v1`

### 1) 登录

请求：

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "<你的_jwt_token>",
    "expires_at": "2026-04-10T09:30:00Z"
  }
}
```

### 2) 分页查询日志（需要 JWT）

```bash
TOKEN='<请粘贴_token>'

curl -s 'http://127.0.0.1:8080/api/v1/logs?page=1&page_size=10' \
  -H "Authorization: Bearer ${TOKEN}"
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "page": 1,
    "page_size": 10,
    "total": 5,
    "items": []
  }
}
```

## 测试

运行全部测试：

```bash
go test ./...
```

当前接口层测试覆盖：

- 登录成功
- 未携带 token 访问日志接口返回未授权
- 携带有效 token 分页查询日志

## SQL 测试素材

用户与日志表的基础 SQL 素材：

- `testdata/sql/user_tables.sql`

这些 SQL 文件可作为测试素材和建表参考。
