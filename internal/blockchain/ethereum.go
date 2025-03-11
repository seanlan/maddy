package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"mailcoin/framework/config"
	"mailcoin/framework/log"
	"mailcoin/framework/module"
	"strings"
)

func verifySignature(message, signature, address string) (bool, error) {
	// 签名为65字节，最后一个字节为v值，需要手动调整
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil {
		return false, err
	}
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}

	// 将 v 值调整为标准值（0或1）
	sigBytes[64] -= 27

	// 对消息进行哈希处理
	msgHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))

	// 恢复公钥
	pubKey, err := crypto.SigToPub(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, err
	}

	// 从公钥导出地址
	recoveredAddress := crypto.PubkeyToAddress(*pubKey).Hex()

	// 比较地址是否一致
	return strings.ToLower(recoveredAddress) == strings.ToLower(address), nil
}

type EVMBlockChain struct {
	modName  string
	instName string
	log      log.Logger

	//TODO add more fields
	chainID int64
	rpcURL  string
}

func (b *EVMBlockChain) SendRawTx(ctx context.Context, rawTx string) error {
	client, err := ethclient.Dial(b.rpcURL)
	if err != nil {
		b.log.Error("failed to dial rpc", err)
		return err
	}
	defer client.Close()
	err = client.Client().CallContext(ctx, nil, "eth_sendRawTransaction", rawTx)
	return err
}

func (b *EVMBlockChain) CheckSign(ctx context.Context, pk, sign, message string) (bool, error) {
	return verifySignature(message, sign, pk)
}

func (b *EVMBlockChain) ChainType(ctx context.Context) string {
	return "ethereum"
}

func NewEVMBlockChain(modName, instName string, _, _ []string) (module.Module, error) {
	return &EVMBlockChain{
		modName:  modName,
		instName: instName,
		log:      log.Logger{Name: modName, Debug: log.DefaultLogger.Debug},
	}, nil
}

func (b *EVMBlockChain) Init(cfg *config.Map) error {
	//TODO implement me
	cfg.Int64("chain_id", false, true, 0, &b.chainID)
	cfg.String("rpc_url", false, true, "", &b.rpcURL)
	if _, err := cfg.Process(); err != nil {
		b.log.Error("failed to process config", err)
		return err
	}
	return nil
}

func (b *EVMBlockChain) Name() string { return b.modName }

func (b *EVMBlockChain) InstanceName() string {
	return b.instName
}

func init() {
	module.Register("blockchain.ethereum", NewEVMBlockChain)
}
