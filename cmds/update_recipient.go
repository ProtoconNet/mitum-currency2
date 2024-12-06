package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/pkg/errors"

	"github.com/ProtoconNet/mitum2/base"
)

type UpdateRecipientCommand struct {
	BaseCommand
	OperationFlags
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Contract   AddressFlag    `arg:"" name:"contract" help:"target contract account address" required:"true"`
	Currency   CurrencyIDFlag `arg:"" name:"currency-id" help:"currency id" required:"true"`
	Recipients []AddressFlag  `arg:"" name:"recipients" help:"recipients"`
	OperationExtensionFlags
	sender      base.Address
	target      base.Address
	didContract base.Address
	proxyPayer  base.Address
	opSender    base.Address
}

func (cmd *UpdateRecipientCommand) Run(pctx context.Context) error {
	if _, err := cmd.prepare(pctx); err != nil {
		return err
	}

	encs = cmd.Encoders
	enc = cmd.Encoder

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

func (cmd *UpdateRecipientCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if len(cmd.Recipients) < 1 {
		return errors.Errorf("Empty recipients, must be given at least one")
	}

	if sender, err := cmd.Sender.Encode(enc); err != nil {
		return errors.Wrapf(err, "invalid sender format, %v", cmd.Sender.String())
	} else if target, err := cmd.Contract.Encode(enc); err != nil {
		return errors.Wrapf(err, "invalid contract address format, %v", cmd.Contract.String())
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

func (cmd *UpdateRecipientCommand) createOperation() (base.Operation, error) { // nolint:dupl
	recipients := make([]base.Address, len(cmd.Recipients))
	for i := range cmd.Recipients {
		ad, err := base.DecodeAddress(cmd.Recipients[i].String(), enc)
		if err != nil {
			return nil, err
		}

		recipients[i] = ad
	}

	fact := extension.NewUpdateRecipientFact([]byte(cmd.Token), cmd.sender, cmd.target, recipients, cmd.Currency.CID)

	op, err := extension.NewUpdateRecipient(fact)
	if err != nil {
		return nil, errors.Wrap(err, "create updateRecipient operation")
	}

	var baseAuthentication common.Authentication
	var baseSettlement common.Settlement
	var proofData = cmd.Proof
	if cmd.IsPrivateKey {
		prk, err := base.DecodePrivatekeyFromString(cmd.Proof, enc)
		if err != nil {
			return nil, err
		}

		sig, err := prk.Sign(fact.Hash().Bytes())
		if err != nil {
			return nil, err
		}
		proofData = sig.String()
	}

	if cmd.didContract != nil && cmd.AuthenticationID != "" && cmd.Proof != "" {
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
