package cmds

import (
	"context"
	did "github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
)

type UpdateDIDDocumentCommand struct {
	BaseCommand
	OperationFlags
	Sender   AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Contract AddressFlag    `arg:"" name:"contract" help:"contract address" required:"true"`
	DID      string         `arg:"" name:"did" help:"did" required:"true"`
	Currency CurrencyIDFlag `arg:"" name:"currency" help:"currency id" required:"true"`
	Document string         `arg:"" name:"document" help:"document; default is stdin" required:"true" default:"-"`
	IsString bool           `name:"document.is-string" help:"input is string, not file"`
	OperationExtensionFlags
	sender   base.Address
	contract base.Address
	document types.DIDDocument
}

func (cmd *UpdateDIDDocumentCommand) Run(pctx context.Context) error { // nolint:dupl
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

func (cmd *UpdateDIDDocumentCommand) parseFlags() error {
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

	if len(cmd.DID) < 1 {
		return errors.Errorf("invalid DID, %s", cmd.DID)
	}

	var doc types.DIDDocument

	switch i, err := launch.LoadInputFlag(cmd.Document, !cmd.IsString); {
	case err != nil:
		return err
	case len(i) < 1:
		return errors.Errorf("Empty document input")
	default:
		cmd.Log.Debug().
			Str("input", string(i)).
			Msg("input")

		if err := encoder.Decode(cmd.Encoder, i, &doc); err != nil {
			return err
		}
	}

	cmd.OperationExtensionFlags.parseFlags(cmd.Encoders.JSON())

	cmd.document = doc

	return nil
}

func (cmd *UpdateDIDDocumentCommand) createOperation() (base.Operation, error) { // nolint:dupl
	e := util.StringError("failed to create issue operation")

	fact := did.NewUpdateDIDDocumentFact([]byte(cmd.Token), cmd.sender, cmd.contract, cmd.DID, cmd.document, cmd.Currency.CID)

	op, err := did.NewUpdateDIDDocument(fact)
	if err != nil {
		return nil, e.Wrap(err)
	}

	var baseAuthentication extras.OperationExtension
	var baseSettlement extras.OperationExtension
	var baseProxyPayer extras.OperationExtension
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
		baseAuthentication = extras.NewBaseAuthentication(cmd.didContract, cmd.AuthenticationID, proofData)
		if err := op.AddExtension(baseAuthentication); err != nil {
			return nil, err
		}
	}

	if cmd.proxyPayer != nil {
		baseProxyPayer = extras.NewBaseProxyPayer(cmd.proxyPayer)
		if err := op.AddExtension(baseProxyPayer); err != nil {
			return nil, err
		}
	}

	if cmd.opSender != nil {
		baseSettlement = extras.NewBaseSettlement(cmd.opSender)
		if err := op.AddExtension(baseSettlement); err != nil {
			return nil, err
		}

		err = op.Sign(cmd.OpSenderPrivatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return nil, errors.Wrapf(err, "create %T operation", op)
		}
	} else {
		err = op.Sign(cmd.Privatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return nil, errors.Wrapf(err, "create %T operation", op)
		}
	}

	if err := op.IsValid(cmd.OperationFlags.NetworkID); err != nil {
		return nil, errors.Wrapf(err, "create %T operation", op)
	}

	return op, nil
}
