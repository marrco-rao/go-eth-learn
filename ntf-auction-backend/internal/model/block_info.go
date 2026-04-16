package model

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockSummaryResponse: 用于列表/范围查询
type BlockSummaryResponse struct {
	Number      uint64  `json:"number"`
	Hash        string  `json:"hash"`
	ParentHash  string  `json:"parent_hash"`
	Timestamp   uint64  `json:"timestamp"`
	Time        string  `json:"time"` // RFC3339
	TxCount     int     `json:"tx_count"`
	GasUsed     uint64  `json:"gas_used"`
	GasLimit    uint64  `json:"gas_limit"`
	GasUsagePct float64 `json:"gas_usage_pct"` // 例如 52.31
}

// BlockDetailResponse: 用于单块详情
type BlockDetailResponse struct {
	Number        uint64   `json:"number"`
	Hash          string   `json:"hash"`
	ParentHash    string   `json:"parent_hash"`
	UncleHash     string   `json:"uncle_hash"`
	Timestamp     uint64   `json:"timestamp"`
	Time          string   `json:"time"` // RFC3339
	Size          uint64   `json:"size"`
	TxCount       int      `json:"tx_count"`
	TxHashes      []string `json:"tx_hashes,omitempty"`
	GasUsed       uint64   `json:"gas_used"`
	GasLimit      uint64   `json:"gas_limit"`
	GasUsagePct   float64  `json:"gas_usage_pct"`
	BaseFeePerGas string   `json:"base_fee_per_gas,omitempty"` // wei，字符串避免精度问题
	Difficulty    string   `json:"difficulty"`                 // 大整数统一字符串
	StateRoot     string   `json:"state_root"`
	TxRoot        string   `json:"tx_root"`
	ReceiptRoot   string   `json:"receipt_root"`
	Coinbase      string   `json:"coinbase,omitempty"`
	FirstTxHash   string   `json:"first_tx_hash,omitempty"`
	LastTxHash    string   `json:"last_tx_hash,omitempty"`
}

// ToBlockSummaryResponse: 区块 -> summary
func ToBlockSummaryResponse(block *types.Block) BlockSummaryResponse {
	if block == nil {
		return BlockSummaryResponse{}
	}

	txCount := len(block.Transactions())
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()

	return BlockSummaryResponse{
		Number:      block.Number().Uint64(),
		Hash:        block.Hash().Hex(),
		ParentHash:  block.ParentHash().Hex(),
		Timestamp:   block.Time(),
		Time:        time.Unix(int64(block.Time()), 0).UTC().Format(time.RFC3339),
		TxCount:     txCount,
		GasUsed:     gasUsed,
		GasLimit:    gasLimit,
		GasUsagePct: calcGasUsagePct(gasUsed, gasLimit),
	}
}

// ToBlockDetailResponse: 区块 -> detail
func ToBlockDetailResponse(block *types.Block) BlockDetailResponse {
	if block == nil {
		return BlockDetailResponse{}
	}

	txs := block.Transactions()
	txCount := len(txs)

	txHashes := make([]string, 0, txCount)
	for _, tx := range txs {
		txHashes = append(txHashes, tx.Hash().Hex())
	}

	resp := BlockDetailResponse{
		Number:      block.Number().Uint64(),
		Hash:        block.Hash().Hex(),
		ParentHash:  block.ParentHash().Hex(),
		UncleHash:   block.UncleHash().Hex(),
		Timestamp:   block.Time(),
		Time:        time.Unix(int64(block.Time()), 0).UTC().Format(time.RFC3339),
		Size:        uint64(block.Size()),
		TxCount:     txCount,
		TxHashes:    txHashes,
		GasUsed:     block.GasUsed(),
		GasLimit:    block.GasLimit(),
		GasUsagePct: calcGasUsagePct(block.GasUsed(), block.GasLimit()),
		Difficulty:  formatBigInt(block.Difficulty()),
		StateRoot:   block.Root().Hex(),
		TxRoot:      block.TxHash().Hex(),
		ReceiptRoot: block.ReceiptHash().Hex(),
	}

	// London 之后有 BaseFee，旧区块可能为 nil
	if baseFee := block.BaseFee(); baseFee != nil {
		resp.BaseFeePerGas = baseFee.String()
	}

	// Coinbase 为零地址时可不返回
	if coinbase := block.Coinbase(); coinbase != (common.Address{}) {
		resp.Coinbase = coinbase.Hex()
	}

	if txCount > 0 {
		resp.FirstTxHash = txs[0].Hash().Hex()
		resp.LastTxHash = txs[txCount-1].Hash().Hex()
	}

	return resp
}

func calcGasUsagePct(gasUsed, gasLimit uint64) float64 {
	if gasLimit == 0 {
		return 0
	}
	// 保留两位小数
	v := (float64(gasUsed) / float64(gasLimit)) * 100
	return float64(int(v*100+0.5)) / 100
}

func formatBigInt(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
