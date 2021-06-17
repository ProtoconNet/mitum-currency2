package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *KeyUpdaterFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	btarget base.AddressDecoder,
	bks []byte,
	cr string,
) error {
	target, err := btarget.Encode(enc)
	if err != nil {
		return err
	}

	var keys Keys
	if hinter, err := enc.Decode(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		keys = k
	}

	fact.h = h
	fact.token = token
	fact.target = target
	fact.keys = keys
	fact.currency = CurrencyID(cr)

	return nil
}
