package modify

import (
	"context"
	"github.com/emersion/go-message/textproto"
	"github.com/dsoftgames/MailChat/framework/buffer"
	"github.com/dsoftgames/MailChat/framework/config"
	modconfig "github.com/dsoftgames/MailChat/framework/config/module"
	"github.com/dsoftgames/MailChat/framework/module"
)

const (
	blockchainRawTxMailHeader = "X-Blockchain-Tx"
	blockchainTypeHeader      = "X-Blockchain-Type"
)

type blockchainTxSender struct {
	modName    string
	instName   string
	inlineArgs []string

	chain module.BlockChain
}

func NewBlockchainTxSender(modName, instName string, _, inlineArgs []string) (module.Module, error) {
	b := blockchainTxSender{
		modName:    modName,
		instName:   instName,
		inlineArgs: inlineArgs,
	}

	return &b, nil
}

func (b *blockchainTxSender) Init(cfg *config.Map) error {
	err := modconfig.ModuleFromNode("blockchain_tx", b.inlineArgs, cfg.Block, cfg.Globals, &b.chain)
	return err
}

func (b *blockchainTxSender) Name() string {
	return b.modName
}

func (b *blockchainTxSender) InstanceName() string {
	return b.instName
}

func (b *blockchainTxSender) ModStateForMsg(ctx context.Context, msgMeta *module.MsgMetadata) (module.ModifierState, error) {
	return b, nil
}

func (b *blockchainTxSender) RewriteSender(ctx context.Context, mailFrom string) (string, error) {
	return mailFrom, nil
}

func (b *blockchainTxSender) RewriteRcpt(ctx context.Context, rcptTo string) ([]string, error) {
	return []string{rcptTo}, nil
}

func (b *blockchainTxSender) RewriteBody(ctx context.Context, h *textproto.Header, body buffer.Buffer) error {
	c, ok := b.chain.(module.BlockChain)
	if !ok {
		return nil
	}
	if c.ChainType(ctx) == h.Get(blockchainTypeHeader) && h.Get(blockchainRawTxMailHeader) != "" {
		return c.SendRawTx(ctx, h.Get(blockchainRawTxMailHeader))
	}
	return nil
}

func (b *blockchainTxSender) Close() error {
	return nil
}

func init() {
	module.Register("modify.blockchain_tx", NewBlockchainTxSender)
}
