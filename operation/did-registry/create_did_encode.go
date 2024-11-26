package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

func (fact *CreateDIDFact) unpack(
	enc encoder.Encoder,
	sa, ta, authType string, publicKey mitumbase.Publickey, svcType, svcEndpoint, cid string,
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

	fact.authType = authType
	fact.publicKey = publicKey
	fact.serviceType = svcType
	fact.serviceEndpoint = svcEndpoint
	fact.currency = types.CurrencyID(cid)

	return nil
}
