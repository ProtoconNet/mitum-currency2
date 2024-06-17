package currency

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum-currency/v3/common"

	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type RegisterGenesisCurrencyFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	GenesisNodeKey base.Publickey         `json:"genesis_node_key"`
	Keys           types.AccountKeys      `json:"keys"`
	Currencies     []types.CurrencyDesign `json:"currencies"`
}

func (fact RegisterGenesisCurrencyFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(RegisterGenesisCurrencyFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		GenesisNodeKey:        fact.genesisNodeKey,
		Keys:                  fact.keys,
		Currencies:            fact.cs,
	})
}

type RegisterGenesisCurrencyFactJSONUnMarshaler struct {
	base.BaseFactJSONUnmarshaler
	GenesisNodeKey string          `json:"genesis_node_key"`
	Keys           json.RawMessage `json:"keys"`
	Currencies     json.RawMessage `json:"currencies"`
}

func (fact *RegisterGenesisCurrencyFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var uf RegisterGenesisCurrencyFactJSONUnMarshaler
	if err := enc.Unmarshal(b, &uf); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(uf.BaseFactJSONUnmarshaler)

	if err := fact.unpack(enc, uf.GenesisNodeKey, uf.Keys, uf.Currencies); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

func (op RegisterGenesisCurrency) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.BaseOperation)
}
