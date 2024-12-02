package common

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

type BaseOperationJSONMarshaler struct {
	MBaseOperationJSONMarshaler
	Authentication Authentication `json:"authentication"`
	Settlement     Settlement     `json:"settlement"`
}

func (op BaseOperation) JSONMarshaler() BaseOperationJSONMarshaler {
	return BaseOperationJSONMarshaler{
		MBaseOperationJSONMarshaler: op.MBaseOperation.JSONMarshaler(),
		Authentication:              op.Authentication,
		Settlement:                  op.Settlement,
	}
}

func (op BaseOperation) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.JSONMarshaler())
}

type BaseOperationJSONUnmarshaler struct {
	Authentication json.RawMessage `json:"authentication"`
	Settlement     json.RawMessage `json:"settlement"`
}

func (op *BaseOperation) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var mbo MBaseOperation

	err := mbo.DecodeJSON(b, enc)
	if err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	op.MBaseOperation = mbo

	var u BaseOperationJSONUnmarshaler

	err = enc.Unmarshal(b, &u)
	if err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	if u.Authentication != nil {
		var ba BaseAuthentication
		err := ba.DecodeJSON(u.Authentication, enc)
		if err != nil {
			return DecorateError(err, ErrDecodeJson, *op)
		}

		if !ba.Equal(BaseAuthentication{}) {
			op.SetAuthentication(ba)
		}
	}

	if u.Settlement != nil {
		var bs BaseSettlement
		err := bs.DecodeJSON(u.Settlement, enc)
		if err != nil {
			return DecorateError(err, ErrDecodeJson, *op)
		}

		if !bs.Equal(BaseSettlement{}) {
			op.SetSettlement(bs)
		}
	}

	return nil
}

type MBaseOperationJSONMarshaler struct {
	Hash  util.Hash   `json:"hash"`
	Fact  base.Fact   `json:"fact"`
	Signs []base.Sign `json:"signs"`
	hint.BaseHinter
}

func (op MBaseOperation) JSONMarshaler() MBaseOperationJSONMarshaler {
	return MBaseOperationJSONMarshaler{
		BaseHinter: op.BaseHinter,
		Hash:       op.h,
		Fact:       op.fact,
		Signs:      op.signs,
	}
}

func (op MBaseOperation) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.JSONMarshaler())
}

type MBaseOperationJSONUnmarshaler struct {
	Hash  valuehash.HashDecoder `json:"hash"`
	Fact  json.RawMessage       `json:"fact"`
	Signs []json.RawMessage     `json:"signs"`
}

func (op *MBaseOperation) decodeJSON(b []byte, enc encoder.Encoder, u *MBaseOperationJSONUnmarshaler) error {
	if err := enc.Unmarshal(b, u); err != nil {
		return ErrValueInvalid.Wrap(err)
	}

	op.h = u.Hash.Hash()

	if err := encoder.Decode(enc, u.Fact, &op.fact); err != nil {
		return ErrValueInvalid.Wrap(err)
	}

	return nil
}

func (op *MBaseOperation) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u MBaseOperationJSONUnmarshaler

	if err := op.decodeJSON(b, enc, &u); err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	op.signs = make([]base.Sign, len(u.Signs))

	for i := range u.Signs {
		var ub base.BaseSign
		if err := ub.DecodeJSON(u.Signs[i], enc); err != nil {
			return DecorateError(errors.Errorf("Decode sign; %v", err), ErrDecodeJson, *op)
		}

		op.signs[i] = ub
	}

	return nil
}

func (op BaseNodeOperation) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.JSONMarshaler())
}

func (op *BaseNodeOperation) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u MBaseOperationJSONUnmarshaler

	if err := op.decodeJSON(b, enc, &u); err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	op.signs = make([]base.Sign, len(u.Signs))

	for i := range u.Signs {
		var ub base.BaseNodeSign
		if err := ub.DecodeJSON(u.Signs[i], enc); err != nil {
			return DecorateError(errors.Errorf("Decode sign; %v", err), ErrDecodeJson, *op)
		}

		op.signs[i] = ub
	}

	return nil
}
