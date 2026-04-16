package model

import "time"

type TxStatus string

const (
	TxStatusCreated     TxStatus = "created"     // 已创建记录，尚未进入签名阶段
	TxStatusSigning     TxStatus = "signing"     // 签名中
	TxStatusBroadcasted TxStatus = "broadcasted" // 已广播到链上节点
	TxStatusPending     TxStatus = "pending"     // 链上待确认（处于内存池或确认数不足）
	TxStatusConfirmed   TxStatus = "confirmed"   // 链上已确认
	TxStatusFailed      TxStatus = "failed"      // 处理失败
)

type TxRecord struct {
	ID uint `gorm:"primaryKey"` // 自增主键 ID

	IdempotencyKey string `gorm:"size:128;uniqueIndex;not null"` // 幂等键，用于防止重复提交
	BizID          string `gorm:"size:128;index"`                // 业务侧关联 ID

	FromAddress string `gorm:"size:64;not null"` // 发送方地址
	ToAddress   string `gorm:"size:64;not null"` // 接收方地址

	ValueWei string `gorm:"size:78;not null"` // 转账金额（单位：Wei）
	DataHex  string `gorm:"type:text"`        // 交易数据（十六进制）

	GasLimit             uint64 `gorm:"not null"` // Gas 限额
	MaxFeePerGasWei      string `gorm:"size:78"`  // EIP-1559 最大总费用（Wei）
	MaxPriorityFeeGasWei string `gorm:"size:78"`  // EIP-1559 小费上限（Wei）

	Nonce  *uint64 `gorm:"index"`         // 链上 nonce（广播后回填）
	TxHash string  `gorm:"size:80;index"` // 交易哈希（广播后回填）

	Status       TxStatus `gorm:"size:32;index;not null"` // 交易状态
	ErrorMessage string   `gorm:"type:text"`              // 失败原因或错误信息
	RetryCount   int      `gorm:"not null;default:0"`     // 重试次数

	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}
