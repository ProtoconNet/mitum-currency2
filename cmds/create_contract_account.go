package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"

	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/pkg/errors"
)

type CreateContractAccountCommand struct {
	BaseCommand
	OperationFlags
	Sender           AddressFlag        `arg:"" name:"sender" help:"sender address" required:"true"`
	Threshold        uint               `help:"threshold for keys (default: ${create_contract_account_threshold})" default:"${create_contract_account_threshold}"` // nolint
	Key              KeyFlag            `name:"key" help:"key for new account (ex: \"<public key>,<weight>\") separator @"`
	Amount           CurrencyAmountFlag `arg:"" name:"currency-amount" help:"amount (ex: \"<currency>,<amount>\")"`
	DIDContract      AddressFlag        `name:"authentication-contract" help:"contract account for authentication"`
	AuthenticationID string             `name:"authentication-id" help:"auth id for authentication"`
	ProofData        string             `name:"authentication-proof-data" help:"proof data for authentication"`
	IsPrivateKey     bool               `name:"is-privatekey" help:"proor-data is private key, not signature"`
	ProxyPayer       AddressFlag        `name:"settlement-proxy-payer" help:"proxy payer account for settlement"`
	sender           base.Address
	didContract      base.Address
	proxyPayer       base.Address
	keys             types.AccountKeys
}

func (cmd *CreateContractAccountCommand) Run(pctx context.Context) error { // nolint:dupl
	if _, err := cmd.prepare(pctx); err != nil {
		return err
	}

	//encs = cmd.Encoders
	//enc = cmd.Encoder

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

func (cmd *CreateContractAccountCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	a, err := cmd.Sender.Encode(cmd.Encoder)
	if err != nil {
		return errors.Wrapf(err, "invalid sender format, %v", cmd.Sender.String())
	}
	cmd.sender = a

	{
		ks := make([]types.AccountKey, len(cmd.Key.Values))
		for i := range cmd.Key.Values {
			ks[i] = cmd.Key.Values[i]
		}

		var kys types.AccountKeys
		if kys, err = types.NewBaseAccountKeys(ks, cmd.Threshold); err != nil {
			return err
		}

		if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	if len(cmd.DIDContract.String()) > 0 {
		a, err := cmd.DIDContract.Encode(cmd.Encoders.JSON())
		if err != nil {
			return errors.Wrapf(err, "invalid contract format, %v", cmd.DIDContract.String())
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

func (cmd *CreateContractAccountCommand) createOperation() (base.Operation, error) { // nolint:dupl}
	var items []extension.CreateContractAccountItem

	ams := make([]types.Amount, 1)
	am := types.NewAmount(cmd.Amount.Big, cmd.Amount.CID)
	if err := am.IsValid(nil); err != nil {
		return nil, err
	}

	ams[0] = am

	item := extension.NewCreateContractAccountItemMultiAmounts(cmd.keys, ams)
	if err := item.IsValid(nil); err != nil {
		return nil, err
	}
	items = append(items, item)

	fact := extension.NewCreateContractAccountFact([]byte(cmd.Token), cmd.sender, items)

	op, err := extension.NewCreateContractAccount(fact)
	if err != nil {
		return nil, errors.Wrap(err, "create create-contract-account operation")
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

	err = op.HashSign(cmd.Privatekey, cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, errors.Wrap(err, "create create-contract-account operation")
	}

	return op, nil
}
