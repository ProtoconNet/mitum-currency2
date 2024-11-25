package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	did "github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type CreateDIDCommand struct {
	BaseCommand
	OperationFlags
	Sender           AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Contract         AddressFlag    `arg:"" name:"contract" help:"contract address" required:"true"`
	AuthType         string         `arg:"" name:"auth-type" help:"authentication type" required:"true"`
	PubKey           string         `arg:"" name:"pubKey" help:"public key" required:"true"`
	ServiceType      string         `arg:"" name:"service-type" help:"service type" required:"true"`
	ServiceEndpoint  string         `arg:"" name:"service-endpoint" help:"service endpoint" required:"true"`
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
	pubKey           base.Publickey
}

func (cmd *CreateDIDCommand) Run(pctx context.Context) error { // nolint:dupl
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

func (cmd *CreateDIDCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	a, err := cmd.Sender.Encode(cmd.Encoders.JSON())
	if err != nil {
		return errors.Wrapf(err, "invalid sender format, %q", cmd.Sender)
	} else {
		cmd.sender = a
	}

	a, err = cmd.Contract.Encode(cmd.Encoders.JSON())
	if err != nil {
		return errors.Wrapf(err, "invalid contract format, %q", cmd.Contract)
	} else {
		cmd.contract = a
	}

	if len(cmd.PubKey) < 1 {
		return errors.Errorf("invalid Public Key, %s", cmd.PubKey)
	}

	switch pk, err := base.DecodePublickeyFromString(cmd.PubKey, enc); {
	case err != nil:
		return err
	default:
		cmd.pubKey = pk
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

func (cmd *CreateDIDCommand) createOperation() (base.Operation, error) { // nolint:dupl
	e := util.StringError("failed to create CreateDID operation")

	fact := did.NewCreateDIDFact(
		[]byte(cmd.Token), cmd.sender, cmd.contract, cmd.sender,
		cmd.AuthType, cmd.pubKey, cmd.ServiceType, cmd.ServiceEndpoint, cmd.Currency.CID,
	)

	op, err := did.NewCreateDID(fact)
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
