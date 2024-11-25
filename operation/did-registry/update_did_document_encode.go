package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

func (fact *UpdateDIDDocumentFact) unpack(
	enc encoder.Encoder,
	sa, ta string,
	did, cid string,
) error {
	switch sender, err := mitumbase.DecodeAddress(sa, enc); {
	case err != nil:
		return err
	default:
		fact.sender = sender
	}

	switch contract, err := mitumbase.DecodeAddress(ta, enc); {
	case err != nil:
		return err
	default:
		fact.contract = contract
	}

	fact.did = did
	fact.currency = types.CurrencyID(cid)

	return nil
}
