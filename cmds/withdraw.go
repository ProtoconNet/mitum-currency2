package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"

	"github.com/pkg/errors"

	"github.com/ProtoconNet/mitum2/base"
)

type WithdrawCommand struct {
	BaseCommand
	OperationFlags
	Sender AddressFlag        `arg:"" name:"sender" help:"sender address" required:"true"`
	Target AddressFlag        `arg:"" name:"target" help:"target contract account address" required:"true"`
	Amount CurrencyAmountFlag `arg:"" name:"currency-amount" help:"amount (ex: \"<currency>,<amount>\")"`
	OperationExtensionFlags
	sender      base.Address
	target      base.Address
	didContract base.Address
	proxyPayer  base.Address
	opSender    base.Address
}

func (cmd *WithdrawCommand) Run(pctx context.Context) error {
	if _, err := cmd.prepare(pctx); err != nil {
		return err
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	op, err := cmd.createOperation()
	if err != nil {
		return err
	}

	PrettyPrint(cmd.Out, op)

	return nil
}

func (cmd *WithdrawCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if sender, err := cmd.Sender.Encode(enc); err != nil {
		return errors.Wrapf(err, "invalid sender format, %v", cmd.Sender.String())
	} else if target, err := cmd.Target.Encode(enc); err != nil {
		return errors.Wrapf(err, "invalid target format, %v", cmd.Target.String())
	} else {
		cmd.sender = sender
		cmd.target = target
	}

	if len(cmd.DIDContract.String()) > 0 {
		a, err := cmd.DIDContract.Encode(cmd.Encoders.JSON())
		if err != nil {
			return errors.Wrapf(err, "invalid did contract format, %v", cmd.DIDContract.String())
		}
		cmd.didContract = a
	}

	if len(cmd.OpSender.String()) > 0 {
		a, err := cmd.OpSender.Encode(cmd.Encoders.JSON())
		if err != nil {
			return errors.Wrapf(err, "invalid proxy payer format, %v", cmd.ProxyPayer.String())
		}
		cmd.opSender = a
	}

	if len(cmd.ProxyPayer.String()) > 0 {
		a, err := cmd.ProxyPayer.Encode(cmd.Encoders.JSON())
		if err != nil {
			return errors.Wrapf(err, "invalid proxy payer format, %v", cmd.ProxyPayer.String())
		}
		cmd.proxyPayer = a
	}

	return nil
}

func (cmd *WithdrawCommand) createOperation() (base.Operation, error) { // nolint:dupl
	var items []extension.WithdrawItem

	ams := make([]types.Amount, 1)
	am := types.NewAmount(cmd.Amount.Big, cmd.Amount.CID)
	if err := am.IsValid(nil); err != nil {
		return nil, err
	}

	ams[0] = am

	item := extension.NewWithdrawItemMultiAmounts(cmd.target, ams)
	if err := item.IsValid(nil); err != nil {
		return nil, err
	}
	items = append(items, item)

	fact := extension.NewWithdrawFact([]byte(cmd.Token), cmd.sender, items)

	op, err := extension.NewWithdraw(fact)
	if err != nil {
		return nil, errors.Wrap(err, "create withdraw operation")
	}

	var baseAuthentication common.Authentication
	var baseSettlement common.Settlement
	var proofData = cmd.ProofData
	if cmd.IsPrivateKey {
		prk, err := base.DecodePrivatekeyFromString(cmd.ProofData, enc)
		if err != nil {
			return nil, err
		}

		sig, err := prk.Sign(fact.Hash().Bytes())
		if err != nil {
			return nil, err
		}
		proofData = sig.String()
	}

	if cmd.didContract != nil && cmd.AuthenticationID != "" && cmd.ProofData != "" {
		baseAuthentication = common.NewBaseAuthentication(cmd.didContract, cmd.AuthenticationID, proofData)
		op.SetAuthentication(baseAuthentication)
	}

	if cmd.opSender != nil {
		baseSettlement = common.NewBaseSettlement(cmd.opSender, cmd.proxyPayer)
		op.SetSettlement(baseSettlement)

		err = op.HashSign(cmd.OpSenderPrivatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return nil, errors.Wrap(err, "create create-account operation")
		}
	} else {
		err = op.HashSign(cmd.Privatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return nil, errors.Wrap(err, "create create-account operation")
		}
	}

	return op, nil
}
