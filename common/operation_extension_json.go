package common

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type BaseAuthenticationJSONMarshaler struct {
	Contract         base.Address `json:"contract"`
	AuthenticationID string       `json:"authentication_id"`
	ProofData        string       `json:"proof_data"`
}

func (op BaseAuthentication) JSONMarshaler() BaseAuthenticationJSONMarshaler {
	return BaseAuthenticationJSONMarshaler{
		Contract:         op.contract,
		AuthenticationID: op.authenticationID,
		ProofData:        op.proofData,
	}
}

func (op BaseAuthentication) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.JSONMarshaler())
}

type BaseAuthenticationJSONUnmarshaler struct {
	Contract         string `json:"contract"`
	AuthenticationID string `json:"authentication_id"`
	ProofData        string `json:"proof_data"`
}

func (op *BaseAuthentication) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u BaseAuthenticationJSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	a, err := base.DecodeAddress(u.Contract, enc)
	if err != nil {
		if err != nil {
			return DecorateError(err, ErrDecodeBson, *op)
		}
	}
	op.contract = a

	op.authenticationID = u.AuthenticationID
	op.proofData = u.ProofData

	return nil
}

type BaseSettlementJSONMarshaler struct {
	ProxyPayer base.Address `json:"proxy_payer"`
}

func (op BaseSettlement) JSONMarshaler() BaseSettlementJSONMarshaler {
	return BaseSettlementJSONMarshaler{
		ProxyPayer: op.proxyPayer,
	}
}

func (op BaseSettlement) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(op.JSONMarshaler())
}

type BaseSettlementJSONUnmarshaler struct {
	ProxyPayer string `json:"proxy_payer"`
}

func (op *BaseSettlement) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u BaseSettlementJSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeJson, *op)
	}

	a, err := base.DecodeAddress(u.ProxyPayer, enc)
	if err != nil {
		if err != nil {
			return DecorateError(err, ErrDecodeJson, *op)
		}
	}
	op.proxyPayer = a

	return nil
}
