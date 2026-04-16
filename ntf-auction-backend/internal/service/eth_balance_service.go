package service

import (
	"context"
	"math"
	"math/big"
	"strings"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/repository"

	"github.com/ethereum/go-ethereum/common"
)

type EthBalanceService struct {
	repo *repository.EthChainRepository
}

type GetETHBalanceInput struct {
	AddressHex  string `json:"addressHex"`
	BlockNumber int64  `json:"blockNumber"`
}

type GetETHBalanceOutput struct {
	AddressHex string `json:"addressHex"`
	BalanceWei string `json:"balanceWei"`
	BalanceEth string `json:"balanceEth"` // 以 ETH 单位表示的余额，保留 6 位小数
}

func NewEthBalanceService(repo *repository.EthChainRepository) *EthBalanceService {
	return &EthBalanceService{repo: repo}
}

func (s *EthBalanceService) GetETHBalance(ctx context.Context, input GetETHBalanceInput) (GetETHBalanceOutput, error) {
	// TODO: 校验地址格式和 blockTag 参数（latest/pending/指定区块高度）。
	addresHex := strings.TrimSpace(input.AddressHex)
	if addresHex == "" {
		return GetETHBalanceOutput{}, apperr.InvalidArgument("addressHex is required")
	}
	if !common.IsHexAddress(addresHex) {
		return GetETHBalanceOutput{}, apperr.InvalidArgument("addressHex format is invalid")
	}

	address := common.HexToAddress(addresHex)

	var blockNum *big.Int
	if input.BlockNumber >= 0 {
		blockNum = big.NewInt(input.BlockNumber)
	} else {
		blockNum = nil // latest
	}

	// 调用 repository 获取
	balanceWei, err := s.repo.GetAccountBalance(ctx, address, blockNum)
	if err != nil {
		return GetETHBalanceOutput{}, apperr.Internal("query balance failed")
	}
	var output = GetETHBalanceOutput{}
	output.AddressHex = addresHex
	balanceEth := weiToEth(balanceWei)
	// 将余额转换为 ETH 单位，并保留 6 位小数，最后以字符串形式输出。这样可以避免浮点数精度问题，同时也符合以太坊中余额的常见表示方式。
	output.BalanceWei = balanceWei.String()
	output.BalanceEth = balanceEth.Text('f', 6)
	return output, nil
}

// weiToEth 将 Wei 转换为 ETH，并保留 6 位小数。
func weiToEth(wei *big.Int) *big.Float {
	fWei := new(big.Float).SetInt(wei)
	ethValue := new(big.Float).Quo(fWei, big.NewFloat(math.Pow10(18)))
	return ethValue
}
