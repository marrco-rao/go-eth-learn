# ntf-auction-backend

基于 **Gin + GORM + go-ethereum + JWT** 的 NFT 拍卖市场后端服务。

## 功能特性

- Gin HTTP 服务，统一响应结构与错误码
- YAML 配置文件 + 环境变量双重驱动
- 数据库支持：SQLite（默认）/ MySQL
- 基于 JWT 的认证机制，所有业务接口受保护
- 请求日志落库到 `logs` 表
- 以太坊区块 / 余额 / 交易查询（go-ethereum）
- 发送交易（EIP-1559，支持幂等键、自动推荐 Gas 参数）
- **链上合约事件订阅**（WebSocket，断线自动重连）：
  - 一次订阅覆盖 9 种 Marketplace 事件
  - 订阅事件列表通过配置文件控制，无需改代码
  - 事件持久化到 `contract_events` 表，支持三维过滤查询

## 项目结构

```text
ntf-auction-backend/
├── abi/                          # abigen 生成的合约绑定
│   ├── marketplace.go
│   └── mynft.go
├── cmd/
│   └── main.go
├── config/
│   └── app.yaml                  # 主配置文件
├── data/
│   └── app.db                    # SQLite 运行时自动生成
├── internal/
│   ├── app/                      # 依赖注入 & 启动入口
│   ├── apperr/                   # 统一错误类型
│   ├── config/                   # Viper 配置加载
│   ├── database/                 # GORM 数据库初始化
│   ├── handler/http/             # HTTP 路由 & 处理器
│   ├── logger/                   # 数据库日志适配器
│   ├── middleware/               # JWT 认证 & 请求日志中间件
│   ├── model/                    # GORM 模型定义
│   ├── repository/               # 数据访问层
│   ├── response/                 # 统一响应封装
│   └── service/                  # 业务逻辑层
└── testdata/sql/
    └── user_tables.sql           # 建表 SQL 参考
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 启动（本地 Anvil 测试节点）

```bash
# 启动 Anvil
anvil

# 设置链上相关环境变量（可选，不设置则仅 HTTP 接口可用）
export ETH_WS_URL=ws://127.0.0.1:8545
export MARKETPLACE_CONTRACT_ADDRESS=0x你的合约地址

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

jwt:
  secret: change-this-secret
  expire_hours: 24

default_user:
  username: admin
  password: admin123

ethereum:
  rpc_url: http://127.0.0.1:8545          # HTTP RPC，应用启动必需
  tx_signer_private_key: ""               # 发送交易时必需，通过环境变量注入
  watch_events:                            # 需要订阅的合约事件，注释掉则不订阅
    - NFTAuctionListed
    - NFTAuctionBidPlaced
    - NFTAuctionEnded
    - NFTAuctionCancelled
    - NFTAuctionDelisted
    - NFTListed
    - NFTDelisted
    - NFTPurchased
    - NFTListingPriceUpdated
```

敏感配置通过环境变量注入（优先级高于配置文件）：

```bash
export ETH_RPC_URL="https://sepolia.infura.io/v3/<your-key>"
export ETH_TX_SIGNER_PRIVATE_KEY="<your-private-key-hex>"
export ETH_WS_URL="wss://sepolia.infura.io/ws/v3/<your-key>"
export MARKETPLACE_CONTRACT_ADDRESS="0x合约地址"
```

## 切换到 MySQL

```yaml
database:
  driver: mysql
  mysql_dsn: your_user:your_password@tcp(127.0.0.1:3306)/ntf_auction?charset=utf8mb4&parseTime=True&loc=Local
```

## 接口列表

基础路径：`/api/v1`，除登录外所有接口需要 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 登录，返回 JWT token |
| GET | `/logs` | 分页查询请求日志 |
| GET | `/blocks/latest` | 查询最新区块 |
| GET | `/blocks/:number` | 按区块号查询区块 |
| GET | `/blocks?start=&end=` | 批量查询区块范围 |
| GET | `/balance/:address` | 查询地址 ETH 余额 |
| GET | `/transactions/:hash` | 按哈希查询链上交易 |
| POST | `/transactions` | 发送交易（EIP-1559） |
| GET | `/contract-events` | 查询已订阅的合约事件 |

### 登录

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "<jwt_token>",
    "expires_at": "2026-04-17T09:00:00Z"
  }
}
```

### 查询合约事件（三维过滤）

```bash
TOKEN='<jwt_token>'

# 查询所有 NFTAuctionListed 事件
curl -s "http://127.0.0.1:8080/api/v1/contract-events?event_type=NFTAuctionListed&page=1&page_size=20" \
  -H "Authorization: Bearer ${TOKEN}"

# 按合约地址 + 事件类型双维过滤
curl -s "http://127.0.0.1:8080/api/v1/contract-events?contract_address=0x...&event_type=NFTPurchased" \
  -H "Authorization: Bearer ${TOKEN}"
```

查询参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `network_ws_url` | string | 可选，按订阅节点 URL 过滤 |
| `contract_address` | string | 可选，按合约地址过滤 |
| `event_type` | string | 可选，按事件名过滤（如 `NFTAuctionListed`） |
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页条数，默认 20 |

### 发送交易

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/transactions \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "idempotencyKey": "tx-001",
    "toAddressHex": "0xRecipientAddress",
    "valueWei": "1000000000000000000",
    "gasLimit": 21000,
    "useSuggestedGasParams": true
  }'
```

## 链上事件订阅说明

- 实现文件：`internal/service/auction_event_subscriber.go`
- 服务启动时，若 `ETH_WS_URL` 和 `MARKETPLACE_CONTRACT_ADDRESS` 均已设置，`app.Run()` 自动在后台 goroutine 启动订阅
- WebSocket 断线后按 `5s` 间隔自动重连
- TxHash + LogIndex 联合唯一索引防止重复入库
- 未配置环境变量时订阅器静默关闭，不影响 HTTP 服务

## 测试

```bash
go test ./...
```

## SQL 参考

所有表的建表 SQL：

- `testdata/sql/user_tables.sql`

包含：`users`、`logs`、`tx_records`、`contract_events` 四张表。
