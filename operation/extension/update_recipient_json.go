package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type UpdateRecipientsFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	Sender     base.Address     `json:"sender"`
	Contract   base.Address     `json:"contract"`
	Recipients []base.Address   `json:"recipients"`
	Currency   types.CurrencyID `json:"currency"`
}

func (fact UpdateRecipientFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(UpdateRecipientsFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Sender:                fact.sender,
		Contract:              fact.contract,
		Recipients:            fact.recipients,
		Currency:              fact.currency,
	})
}

type UpdatRecipientsFactJSONUnMarshaler struct {
	base.BaseFactJSONUnmarshaler
	Sender     string   `json:"sender"`
	Contract   string   `json:"contract"`
	Recipients []string `json:"recipients"`
	Currency   string   `json:"currency"`
}

func (fact *UpdateRecipientFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var uf UpdatRecipientsFactJSONUnMarshaler
	if err := enc.Unmarshal(b, &uf); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(uf.BaseFactJSONUnmarshaler)

	if err := fact.unpack(enc, uf.Sender, uf.Contract, uf.Recipients, uf.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

func (op UpdateRecipient) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(currency.BaseOperationMarshaler{
		BaseOperationJSONMarshaler: op.BaseOperation.JSONMarshaler(),
	})
}

func (op *UpdateRecipient) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var ubo common.BaseOperation
	if err := ubo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperation = ubo

	return nil
}
