package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type UpdateOperatorFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	Sender    base.Address     `json:"sender"`
	Contract  base.Address     `json:"contract"`
	Operators []base.Address   `json:"operators"`
	Currency  types.CurrencyID `json:"currency"`
}

func (fact UpdateOperatorFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(UpdateOperatorFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Sender:                fact.sender,
		Contract:              fact.contract,
		Operators:             fact.operators,
		Currency:              fact.currency,
	})
}

type UpdatOperatorFactJSONUnMarshaler struct {
	base.BaseFactJSONUnmarshaler
	Sender    string   `json:"sender"`
	Contract  string   `json:"contract"`
	Operators []string `json:"operators"`
	Currency  string   `json:"currency"`
}

func (fact *UpdateOperatorFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var uf UpdatOperatorFactJSONUnMarshaler
	if err := enc.Unmarshal(b, &uf); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(uf.BaseFactJSONUnmarshaler)

	if err := fact.unpack(enc, uf.Sender, uf.Contract, uf.Operators, uf.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

func (op UpdateOperator) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(currency.BaseOperationMarshaler{
		BaseOperationJSONMarshaler: op.BaseOperation.JSONMarshaler(),
	})
}

func (op *UpdateOperator) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var ubo common.BaseOperation
	if err := ubo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperation = ubo

	return nil
}
