package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	mitumbase "github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type DeactivateDIDFactJSONMarshaler struct {
	mitumbase.BaseFactJSONMarshaler
	Sender   mitumbase.Address `json:"sender"`
	Contract mitumbase.Address `json:"contract"`
	DID      string            `json:"did"`
	Currency types.CurrencyID  `json:"currency"`
}

func (fact DeactivateDIDFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(DeactivateDIDFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Sender:                fact.sender,
		Contract:              fact.contract,
		DID:                   fact.did,
		Currency:              fact.currency,
	})
}

type DeactivateDIDFactJSONUnmarshaler struct {
	mitumbase.BaseFactJSONUnmarshaler
	Sender   string `json:"sender"`
	Contract string `json:"contract"`
	DID      string `json:"did"`
	Currency string `json:"currency"`
}

func (fact *DeactivateDIDFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u DeactivateDIDFactJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(u.BaseFactJSONUnmarshaler)

	if err := fact.unpack(enc, u.Sender, u.Contract, u.DID, u.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

func (op DeactivateDID) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(OperationMarshaler{
		BaseOperationJSONMarshaler:           op.BaseOperation.JSONMarshaler(),
		BaseOperationExtensionsJSONMarshaler: op.BaseOperationExtensions.JSONMarshaler(),
	})
}

func (op *DeactivateDID) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var ubo common.BaseOperation
	if err := ubo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperation = ubo

	var ueo extras.BaseOperationExtensions
	if err := ueo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperationExtensions = &ueo

	return nil
}
