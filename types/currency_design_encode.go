package types

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

func (de *CurrencyDesign) unpack(enc encoder.Encoder, ht hint.Hint, bis []byte, ga string, bpo []byte, ts string) error {
	de.BaseHinter = hint.NewBaseHinter(ht)

	var initialSupply Amount
	if err := encoder.Decode(enc, bis, &initialSupply); err != nil {
		return errors.Errorf("Decode amount, %v", err)
	}

	de.initialSupply = initialSupply

	switch ad, err := base.DecodeAddress(ga, enc); {
	case err != nil:
		return errors.Errorf("Decode address, %v", err)
	default:
		de.genesisAccount = ad
	}

	var policy CurrencyPolicy

	if err := encoder.Decode(enc, bpo, &policy); err != nil {
		return errors.Errorf("Decode currency policy, %v", err)
	}

	de.policy = policy

	if big, err := common.NewBigFromString(ts); err != nil {
		return err
	} else {
		de.totalSupply = big
	}

	return nil
}
