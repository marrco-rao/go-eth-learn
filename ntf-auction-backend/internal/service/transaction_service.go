package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

type TransactionService struct {
	repo                *repository.EthChainRepository
	txRecordRepo        repository.TxRecordRepository
	signerPrivateKeyHex string // 交易签名私钥（十六进制）
}

type GetTransactionInput struct {
	TxHashHex string `json:"txHash"`
}

type GetTransactionOutput struct {
	TxHash          string `json:"txHash"`          // 交易哈希值
	State           string `json:"state"`           // 交易状态（pending/mined_receipt_unavailable/confirmed）
	From            string `json:"from"`            // 发送者以太坊地址
	To              string `json:"to"`              // 接收者以太坊地址
	ValueWei        string `json:"valueWei"`        // 交易金额（单位：Wei）
	Gas             uint64 `json:"gas"`             // 交易燃气限制
	GasPrice        string `json:"gasPrice"`        // 交易燃气价格（单位：Wei）
	Nonce           uint64 `json:"nonce"`           // 交易序号
	DataLen         int    `json:"dataLen"`         // 交易数据长度
	BlockNumber     uint64 `json:"blockNumber"`     // 交易所在区块号
	Confirmations   uint64 `json:"confirmations"`   // 交易确认数
	ReceiptReady    bool   `json:"receiptReady"`    // 交易收据是否已就绪
	Success         bool   `json:"success"`         // 交易是否执行成功
	ReceiptStatus   uint64 `json:"receiptStatus"`   // 交易收据状态（0表示失败，1表示成功）
	GasUsed         uint64 `json:"gasUsed"`         // 交易实际消耗的燃气量
	EffectiveGasWei string `json:"effectiveGasWei"` // 实际燃气价格（单位：Wei）
}

const (
	TxStatePending                 = "pending"
	TxStateMinedReceiptUnavailable = "mined_receipt_unavailable"
	TxStateConfirmed               = "confirmed"
)

type SendTransactionInput struct {
	IdempotencyKey        string `json:"idempotencyKey"`        // 幂等键，用于防止重复提交
	BizID                 string `json:"bizId"`                 // 业务ID，用于关联业务数据
	ToAddressHex          string `json:"toAddressHex"`          // 接收者以太坊地址（十六进制格式）
	ValueWei              string `json:"valueWei"`              // 转账金额（单位：Wei）
	DataHex               string `json:"dataHex"`               // 交易数据（十六进制格式，例如合约调用数据）
	GasLimit              uint64 `json:"gasLimit"`              // 燃气限制
	MaxFeePerGasWei       string `json:"maxFeePerGasWei"`       // 最大燃气价格（EIP-1559，单位：Wei）
	MaxPriorityFeeGasWei  string `json:"maxPriorityFeeGasWei"`  // 最大优先级燃气费用（EIP-1559，单位：Wei）
	UseSuggestedGasParams bool   `json:"useSuggestedGasParams"` // 是否使用建议的燃气参数
}

type SendTransactionOutput struct {
	TxHash string `json:"txHash"`
}

// NewTransactionService 创建交易服务，并注入链上仓储与交易记录仓储。
func NewTransactionService(repo *repository.EthChainRepository, txRecordRepo repository.TxRecordRepository, signerPrivateKeyHex string) *TransactionService {
	return &TransactionService{
		repo:                repo,
		txRecordRepo:        txRecordRepo,
		signerPrivateKeyHex: signerPrivateKeyHex,
	}
}

// GetTransaction 按交易哈希查询链上交易与回执，并组装统一返回结构。
func (s *TransactionService) GetTransaction(ctx context.Context, input GetTransactionInput) (GetTransactionOutput, error) {
	txHashHex := strings.TrimSpace(input.TxHashHex)
	if txHashHex == "" {
		return GetTransactionOutput{}, apperr.InvalidArgument("txHash is required")
	}
	if !common.IsHexHash(txHashHex) {
		return GetTransactionOutput{}, apperr.InvalidArgument("txHash format is invalid")
	}

	txHash := common.HexToHash(txHashHex)
	tx, receipt, err := s.repo.GetTransactionByHash(ctx, txHash)
	if err != nil && tx == nil {
		if errors.Is(err, ethereum.NotFound) {
			return GetTransactionOutput{}, apperr.NotFound("transaction not found")
		}
		return GetTransactionOutput{}, apperr.Internal("query transaction failed")
	}

	out := buildBaseTransactionOutput(txHash, tx)

	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			out.State = TxStateMinedReceiptUnavailable
			return out, nil
		}
		return GetTransactionOutput{}, apperr.Internal("query transaction receipt failed")
	}

	if receipt == nil {
		out.State = TxStatePending
		return out, nil
	}

	out.State = TxStateConfirmed
	out.ReceiptReady = true
	out.BlockNumber = receipt.BlockNumber.Uint64()
	out.ReceiptStatus = receipt.Status
	out.Success = receipt.Status == types.ReceiptStatusSuccessful
	out.GasUsed = receipt.GasUsed

	if receipt.EffectiveGasPrice != nil {
		out.EffectiveGasWei = receipt.EffectiveGasPrice.String()
	} else if tx.GasPrice() != nil {
		out.EffectiveGasWei = tx.GasPrice().String()
	}

	latest, latestErr := s.repo.GetLatestBlockNumber(ctx)
	if latestErr == nil && latest >= out.BlockNumber {
		out.Confirmations = latest - out.BlockNumber + 1
	}

	return out, nil
}

// buildBaseTransactionOutput 从交易对象提取基础字段，用于统一响应结构。
func buildBaseTransactionOutput(txHash common.Hash, tx *types.Transaction) GetTransactionOutput {
	if tx == nil {
		return GetTransactionOutput{TxHash: txHash.Hex()}
	}

	out := GetTransactionOutput{
		TxHash:   tx.Hash().Hex(),
		ValueWei: tx.Value().String(),
		Nonce:    tx.Nonce(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice().String(),
		DataLen:  len(tx.Data()),
	}

	if to := tx.To(); to != nil {
		out.To = to.Hex()
	}

	if from, err := extractSender(tx); err == nil {
		out.From = from.Hex()
	}

	return out
}

// extractSender 尝试根据不同签名器恢复交易发送方地址。
func extractSender(tx *types.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, errors.New("transaction is nil")
	}

	chainID := tx.ChainId()
	if chainID != nil && chainID.Sign() > 0 {
		from, err := types.Sender(types.LatestSignerForChainID(chainID), tx)
		if err != nil {
			return common.Address{}, fmt.Errorf("recover sender with chain signer failed: %w", err)
		}
		return from, nil
	}

	if from, err := types.Sender(types.HomesteadSigner{}, tx); err == nil {
		return from, nil
	}

	if from, err := types.Sender(types.FrontierSigner{}, tx); err == nil {
		return from, nil
	}

	return common.Address{}, errors.New("recover sender failed with legacy signers")
}

// SendTransaction 完成幂等校验、签名、广播和交易记录落库。
func (s *TransactionService) SendTransaction(ctx context.Context, input SendTransactionInput) (SendTransactionOutput, error) {
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		return SendTransactionOutput{}, apperr.InvalidArgument("idempotencyKey is required")
	}

	if strings.TrimSpace(input.ToAddressHex) == "" {
		return SendTransactionOutput{}, apperr.InvalidArgument("toAddressHex is required")
	}
	if !common.IsHexAddress(strings.TrimSpace(input.ToAddressHex)) {
		return SendTransactionOutput{}, apperr.InvalidArgument("toAddressHex format is invalid")
	}

	valueWei, err := parseWei(input.ValueWei)
	if err != nil {
		return SendTransactionOutput{}, apperr.InvalidArgument("valueWei format is invalid")
	}

	data, err := parseHexData(input.DataHex)
	if err != nil {
		return SendTransactionOutput{}, apperr.InvalidArgument("dataHex format is invalid")
	}

	if strings.TrimSpace(s.signerPrivateKeyHex) == "" {
		return SendTransactionOutput{}, apperr.Internal("tx signer configuration is missing")
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(s.signerPrivateKeyHex), "0x"))
	if err != nil {
		return SendTransactionOutput{}, apperr.Internal("tx signer private key is invalid")
	}

	signerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	existing, err := s.txRecordRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		if existing.TxHash != "" {
			return SendTransactionOutput{TxHash: existing.TxHash}, nil
		}
		return SendTransactionOutput{}, apperr.InvalidArgument("duplicated idempotencyKey: transaction is already processing")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return SendTransactionOutput{}, apperr.Internal("query tx record failed")
	}

	record := model.TxRecord{
		IdempotencyKey:       idempotencyKey,
		BizID:                strings.TrimSpace(input.BizID),
		FromAddress:          signerAddr.Hex(),
		ToAddress:            input.ToAddressHex,
		ValueWei:             input.ValueWei,
		DataHex:              input.DataHex,
		GasLimit:             input.GasLimit,
		MaxFeePerGasWei:      input.MaxFeePerGasWei,
		MaxPriorityFeeGasWei: input.MaxPriorityFeeGasWei,
		Status:               model.TxStatusSigning,
	}
	if err := s.txRecordRepo.Create(ctx, &record); err != nil {
		return SendTransactionOutput{}, apperr.Internal("create tx record failed")
	}

	chainID, err := s.repo.GetChainID(ctx)
	if err != nil {
		_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, err.Error())
		return SendTransactionOutput{}, apperr.Internal("query chain id failed")
	}

	nonce, err := s.repo.GetPendingNonceAt(ctx, signerAddr)
	if err != nil {
		_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, err.Error())
		return SendTransactionOutput{}, apperr.Internal("query account nonce failed")
	}

	maxFeePerGas, maxPriorityFeePerGas, err := s.resolveFeeCaps(ctx, input)
	if err != nil {
		_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, err.Error())
		if input.UseSuggestedGasParams {
			return SendTransactionOutput{}, apperr.Internal(err.Error())
		}
		return SendTransactionOutput{}, apperr.InvalidArgument(err.Error())
	}

	toAddr := common.HexToAddress(strings.TrimSpace(input.ToAddressHex))
	gasLimit := input.GasLimit
	if gasLimit == 0 {
		estimatedGas, estimateErr := s.repo.EstimateGas(ctx, ethereum.CallMsg{
			From:      signerAddr,
			To:        &toAddr,
			GasFeeCap: maxFeePerGas,
			GasTipCap: maxPriorityFeePerGas,
			Value:     valueWei,
			Data:      data,
		})
		if estimateErr != nil {
			_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, estimateErr.Error())
			return SendTransactionOutput{}, apperr.Internal("estimate gas failed")
		}
		gasLimit = estimatedGas
	}
	// 不做余额校验
	unsignedTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      data,
	})

	signedTx, err := types.SignTx(unsignedTx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, err.Error())
		return SendTransactionOutput{}, apperr.Internal("sign transaction failed")
	}

	if err := s.repo.SendTransaction(ctx, signedTx); err != nil {
		_ = s.txRecordRepo.UpdateStatus(ctx, record.ID, model.TxStatusFailed, err.Error())
		return SendTransactionOutput{}, apperr.Internal("broadcast transaction failed")
	}

	txHash := signedTx.Hash().Hex()
	_ = s.txRecordRepo.UpdateBroadcastResult(ctx, record.ID, txHash, nonce)

	return SendTransactionOutput{TxHash: txHash}, nil
}

// parseWei 将十进制或十六进制字符串解析为 Wei 数值。
func parseWei(raw string) (*big.Int, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return big.NewInt(0), nil
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		hexBig, err := hexutil.DecodeBig(v)
		if err != nil {
			return nil, err
		}
		if hexBig.Sign() < 0 {
			return nil, errors.New("negative wei is not allowed")
		}
		return hexBig, nil
	}
	parsed, ok := new(big.Int).SetString(v, 10)
	if !ok {
		return nil, errors.New("invalid decimal wei")
	}
	if parsed.Sign() < 0 {
		return nil, errors.New("negative wei is not allowed")
	}
	return parsed, nil
}

// parseHexData 将十六进制字符串解析为交易 data 字节数组。
func parseHexData(raw string) ([]byte, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		return hexutil.Decode(v)
	}
	return hex.DecodeString(v)
}

// resolveFeeCaps 解析或生成 EIP-1559 所需的 gas fee cap 与 tip cap。
func (s *TransactionService) resolveFeeCaps(ctx context.Context, input SendTransactionInput) (*big.Int, *big.Int, error) {
	if input.UseSuggestedGasParams {
		tipCap, err := s.repo.SuggestGasTipCap(ctx)
		if err != nil {
			return nil, nil, errors.New("suggest gas tip cap failed")
		}

		header, err := s.repo.GetLatestHeader(ctx)
		if err != nil {
			return nil, nil, errors.New("query latest block header failed")
		}

		baseFee := big.NewInt(0)
		if header != nil && header.BaseFee != nil {
			baseFee = header.BaseFee
		}

		maxFeePerGas := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tipCap)
		return maxFeePerGas, tipCap, nil
	}

	maxFeePerGas, err := parseWei(input.MaxFeePerGasWei)
	if err != nil {
		return nil, nil, errors.New("maxFeePerGasWei format is invalid")
	}
	maxPriorityFeePerGas, err := parseWei(input.MaxPriorityFeeGasWei)
	if err != nil {
		return nil, nil, errors.New("maxPriorityFeeGasWei format is invalid")
	}
	if maxFeePerGas.Sign() <= 0 {
		return nil, nil, errors.New("maxFeePerGasWei must be greater than 0")
	}
	if maxPriorityFeePerGas.Sign() <= 0 {
		return nil, nil, errors.New("maxPriorityFeeGasWei must be greater than 0")
	}
	if maxFeePerGas.Cmp(maxPriorityFeePerGas) < 0 {
		return nil, nil, errors.New("maxFeePerGasWei must be greater than or equal to maxPriorityFeeGasWei")
	}

	return maxFeePerGas, maxPriorityFeePerGas, nil
}
