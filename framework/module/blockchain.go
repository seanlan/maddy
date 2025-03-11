package module

import (
	"context"
)

type BlockChain interface {
	// SendRawTx 发送交易
	SendRawTx(ctx context.Context, rawTx string) error
	ChainType(ctx context.Context) string
	CheckSign(ctx context.Context, pk, sign, message string) (bool, error)
}
