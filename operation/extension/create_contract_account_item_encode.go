package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

func (it *BaseCreateContractAccountItem) unpack(enc encoder.Encoder, ht hint.Hint, bks []byte, bam []byte) error {
	it.BaseHinter = hint.NewBaseHinter(ht)

	if hinter, err := enc.Decode(bks); err != nil {
		return err
	} else if k, ok := hinter.(types.AccountKeys); !ok {
		return common.ErrTypeMismatch.Wrap(errors.Errorf("expected AccountsKeys, not %T", hinter))
	} else {
		it.keys = k
	}

	ham, err := enc.DecodeSlice(bam)
	if err != nil {
		return err
	}

	amounts := make([]types.Amount, len(ham))
	for i := range ham {
		j, ok := ham[i].(types.Amount)
		if !ok {
			return common.ErrTypeMismatch.Wrap(errors.Errorf("expected Amount, not %T", ham[i]))
		}

		amounts[i] = j
	}

	it.amounts = amounts

	return nil
}
