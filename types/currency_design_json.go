package types

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

type CurrencyDesignJSONMarshaler struct {
	hint.BaseHinter
	InitialSupply Amount         `json:"initial_supply"`
	Genesis       base.Address   `json:"genesis_account"`
	Policy        CurrencyPolicy `json:"policy"`
	TotalSupply   string         `json:"total_supply"`
}

func (de CurrencyDesign) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(CurrencyDesignJSONMarshaler{
		BaseHinter:    de.BaseHinter,
		InitialSupply: de.initialSupply,
		Genesis:       de.genesisAccount,
		Policy:        de.policy,
		TotalSupply:   de.totalSupply.String(),
	})
}

type CurrencyDesignJSONUnmarshaler struct {
	Hint          hint.Hint       `json:"_hint"`
	InitialSupply json.RawMessage `json:"initial_supply"`
	Genesis       string          `json:"genesis_account"`
	Policy        json.RawMessage `json:"policy"`
	TotalSupply   string          `json:"total_supply"`
}

func (de *CurrencyDesign) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json of CurrencyDesign")

	var ude CurrencyDesignJSONUnmarshaler
	if err := enc.Unmarshal(b, &ude); err != nil {
		return e.Wrap(err)
	}

	return de.unpack(enc, ude.Hint, ude.InitialSupply, ude.Genesis, ude.Policy, ude.TotalSupply)
}
