package cmds

import (
	"context"

	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/pkg/errors"
)

//type MintItemFlag struct {
//	s        string
//	receiver base.Address
//	amount   types.Amount
//}
//
//func (v *MintItemFlag) String() string {
//	return v.s
//}
//
//func (v *MintItemFlag) UnmarshalText(b []byte) error {
//	v.s = string(b)
//
//	l := strings.SplitN(string(b), ",", 3)
//	if len(l) != 3 {
//		return util.ErrInvalid.Errorf("invalid inflation amount, %v", string(b))
//	}
//
//	a, c := l[0], l[1]+","+l[2]
//
//	af := &AddressFlag{}
//	if err := af.UnmarshalText([]byte(a)); err != nil {
//		return util.ErrInvalid.Errorf("invalid inflation receiver address: %v", err)
//	}
//
//	receiver, err := af.Encode(enc)
//	if err != nil {
//		return util.ErrInvalid.Errorf("invalid inflation receiver address: %v", err)
//	}
//
//	v.receiver = receiver
//
//	cf := &CurrencyAmountFlag{}
//	if err := cf.UnmarshalText([]byte(c)); err != nil {
//		return util.ErrInvalid.Errorf("invalid inflation amount: %v", err)
//	}
//	v.amount = types.NewAmount(cf.Big, cf.CID)
//
//	return nil
//}

//func (v *MintItemFlag) IsValid([]byte) error {
//	if err := util.CheckIsValiders(nil, false, v.receiver, v.amount); err != nil {
//		return err
//	}
//
//	if !v.amount.Big().OverZero() {
//		return util.ErrInvalid.Errorf("amount should be over zero")
//	}
//
//	return nil
//}

type MintCommand struct {
	BaseCommand
	OperationFlags
	Node           AddressFlag `arg:"" name:"node" help:"node address" required:"true"`
	node           base.Address
	ReceiverAmount AddressCurrencyAmountFlag `arg:"" name:"receiver amount" help:"ex: \"<receiver address>,<currency>,<amount>\" separator @"`
}

func (cmd *MintCommand) Run(pctx context.Context) error { // nolint:dupl
	if _, err := cmd.prepare(pctx); err != nil {
		return err
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op base.Operation
	if i, err := cmd.createOperation(); err != nil {
		return errors.Wrap(err, "create mint operation")
	} else if err := i.IsValid(cmd.OperationFlags.NetworkID); err != nil {
		return errors.Wrap(err, "invalid mint operation")
	} else {
		cmd.Log.Debug().Interface("operation", i).Msg("operation loaded")

		op = i
	}

	PrettyPrint(cmd.Out, op)

	return nil
}

func (cmd *MintCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	a, err := cmd.Node.Encode(enc)
	if err != nil {
		return errors.Wrapf(err, "invalid node format, %v", cmd.Node.String())
	}
	cmd.node = a

	return nil
}

func (cmd *MintCommand) createOperation() (currency.Mint, error) {
	items := make([]currency.MintItem, len(cmd.ReceiverAmount.Address()))
	for i := range cmd.ReceiverAmount.Address() {
		items[i] = currency.NewMintItem(cmd.ReceiverAmount.Address()[i], cmd.ReceiverAmount.Amount()[i])

		cmd.Log.Debug().
			Stringer("amount", cmd.ReceiverAmount.Amount()[i]).
			Stringer("receiver", cmd.ReceiverAmount.Address()[i]).
			Msg("mint item loaded")
	}

	fact := currency.NewMintFact([]byte(cmd.Token), items)

	op, err := currency.NewMint(fact)
	if err != nil {
		return currency.Mint{}, err
	}

	err = op.NodeSign(cmd.Privatekey, cmd.NetworkID.NetworkID(), cmd.node)
	if err != nil {
		return currency.Mint{}, errors.Wrap(err, "create mint operation")
	}

	return op, nil
}
