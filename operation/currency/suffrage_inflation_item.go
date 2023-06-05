package currency

import (
	base3 "github.com/ProtoconNet/mitum-currency/v3/base"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

var (
	SuffrageInflationItemHint = hint.MustNewHint("mitum-currency-suffrage-inflation-item-v0.0.1")
)

type SuffrageInflationItem struct {
	hint.BaseHinter
	receiver base.Address
	amount   base3.Amount
}

func NewSuffrageInflationItem(receiver base.Address, amount base3.Amount) SuffrageInflationItem {
	return SuffrageInflationItem{
		BaseHinter: hint.NewBaseHinter(SuffrageInflationItemHint),
		receiver:   receiver,
		amount:     amount,
	}
}

func (it SuffrageInflationItem) Bytes() []byte {
	var br []byte
	if it.receiver != nil {
		br = it.receiver.Bytes()
	}

	return util.ConcatBytesSlice(br, it.amount.Bytes())
}

func (it SuffrageInflationItem) Receiver() base.Address {
	return it.receiver
}

func (it SuffrageInflationItem) Currency() base3.CurrencyID {
	return it.amount.Currency()
}

func (it SuffrageInflationItem) Amount() base3.Amount {
	return it.amount
}

func (it SuffrageInflationItem) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, it.receiver, it.amount); err != nil {
		return err
	}

	if !it.amount.Big().OverZero() {
		return util.ErrInvalid.Errorf("under zero amount of SuffrageInflationItemo")
	}

	return nil
}
