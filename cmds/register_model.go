package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"

	did "github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type RegisterModelCommand struct {
	BaseCommand
	OperationFlags
	Sender           AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Contract         AddressFlag    `arg:"" name:"contract" help:"contract account to register policy" required:"true"`
	DIDMethod        string         `arg:"" name:"did-method" help:"did method" required:"true"`
	Currency         CurrencyIDFlag `arg:"" name:"currency" help:"currency id" required:"true"`
	DIDContract      AddressFlag    `name:"authentication-contract" help:"contract account for authentication"`
	AuthenticationID string         `name:"authentication-id" help:"auth id for authentication"`
	ProofData        string         `name:"authentication-proof-data" help:"proof data for authentication"`
	IsPrivateKey     bool           `name:"is-privatekey" help:"proor-data is private key, not signature"`
	ProxyPayer       AddressFlag    `name:"settlement-proxy-payer" help:"proxy payer account for settlement"`
	sender           base.Address
	contract         base.Address
	didContract      base.Address
	proxyPayer       base.Address
}

func (cmd *RegisterModelCommand) Run(pctx context.Context) error {
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

func (cmd *RegisterModelCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if a, err := cmd.Sender.Encode(cmd.Encoders.JSON()); err != nil {
		return errors.Wrapf(err, "invalid sender format; %q", cmd.Sender)
	} else {
		cmd.sender = a
	}

	if a, err := cmd.Contract.Encode(cmd.Encoders.JSON()); err != nil {
		return errors.Wrapf(err, "invalid contract format; %q", cmd.Contract)
	} else {
		cmd.contract = a
	}

	if len(cmd.DIDMethod) < 1 {
		return errors.Errorf("invalid DID Method, %s", cmd.DIDMethod)
	}

	if len(cmd.DIDContract.String()) > 0 {
		a, err := cmd.DIDContract.Encode(cmd.Encoders.JSON())
		if err != nil {
			return errors.Wrapf(err, "invalid did contract format, %v", cmd.DIDContract.String())
		}
		cmd.didContract = a
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

func (cmd *RegisterModelCommand) createOperation() (base.Operation, error) {
	e := util.StringError("failed to create register-model operation")

	fact := did.NewRegisterModelFact(
		[]byte(cmd.Token), cmd.sender, cmd.contract, cmd.DIDMethod, cmd.Currency.CID,
	)

	op, err := did.NewRegisterModel(fact)
	if err != nil {
		return nil, e.Wrap(err)
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
	if cmd.proxyPayer != nil {
		baseSettlement = common.NewBaseSettlement(cmd.proxyPayer)
		op.SetSettlement(baseSettlement)
	}

	err = op.Sign(cmd.Privatekey, cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, e.Wrap(err)
	}

	return op, nil
}
