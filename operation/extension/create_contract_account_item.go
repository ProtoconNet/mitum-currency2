package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

type BaseCreateContractAccountItem struct {
	hint.BaseHinter
	keys    types.AccountKeys
	amounts []types.Amount
}

func NewBaseCreateContractAccountItem(ht hint.Hint, keys types.AccountKeys, amounts []types.Amount) BaseCreateContractAccountItem {
	return BaseCreateContractAccountItem{
		BaseHinter: hint.NewBaseHinter(ht),
		keys:       keys,
		amounts:    amounts,
	}
}

func (it BaseCreateContractAccountItem) Bytes() []byte {
	length := 1
	bs := make([][]byte, len(it.amounts)+1)
	bs[0] = it.keys.Bytes()
	for i := range it.amounts {
		bs[i+length] = it.amounts[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (it BaseCreateContractAccountItem) IsValid([]byte) error {
	if len(it.amounts) < 1 {
		return common.ErrArrayLen.Wrap(errors.Errorf("empty amounts"))
	}

	if err := util.CheckIsValiders(nil, false, it.BaseHinter, it.keys); err != nil {
		return err
	}

	founds := map[types.CurrencyID]struct{}{}
	for i := range it.amounts {
		am := it.amounts[i]
		if _, found := founds[am.Currency()]; found {
			return common.ErrDupVal.Wrap(errors.Errorf("currency id, %v", am.Currency()))
		}
		founds[am.Currency()] = struct{}{}

		if err := am.IsValid(nil); err != nil {
			return err
		} else if !am.Big().OverZero() {
			return common.ErrValOOR.Wrap(errors.Errorf("amount should be over zero"))
		}
	}

	return nil
}

func (it BaseCreateContractAccountItem) Keys() types.AccountKeys {
	return it.keys
}

func (it BaseCreateContractAccountItem) Address() (base.Address, error) {
	return types.NewAddressFromKeys(it.keys)
}

func (it BaseCreateContractAccountItem) Amounts() []types.Amount {
	return it.amounts
}

func (it BaseCreateContractAccountItem) Rebuild() CreateContractAccountItem {
	ams := make([]types.Amount, len(it.amounts))
	for i := range it.amounts {
		am := it.amounts[i]
		ams[i] = am.WithBig(am.Big())
	}

	it.amounts = ams

	return it
}
