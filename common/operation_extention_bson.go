package common

import (
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"go.mongodb.org/mongo-driver/bson"
)

type BaseAuthenticationBSONUnmarshaler struct {
	Contract         string `bson:"contract"`
	AuthenticationID string `bson:"authentication_id"`
	ProofData        string `bson:"proof_data"`
}

func (op BaseAuthentication) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"contract":          op.contract.String(),
			"authentication_id": op.authenticationID,
			"proof_data":        op.proofData,
		},
	)
}

func (op *BaseAuthentication) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	if len(b) < 1 {
		op.contract = nil
		op.authenticationID = ""
		op.proofData = ""

		return nil
	}
	var u BaseAuthenticationBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
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

type BaseSettlementBSONUnmarshaler struct {
	ProxyPayer string `bson:"proxy_payer"`
}

func (op BaseSettlement) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"proxy_payer": op.proxyPayer.String(),
		},
	)
}

func (op *BaseSettlement) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	if len(b) < 1 {
		op.proxyPayer = nil

		return nil
	}
	var u BaseSettlementBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	a, err := base.DecodeAddress(u.ProxyPayer, enc)
	if err != nil {
		if err != nil {
			return DecorateError(err, ErrDecodeBson, *op)
		}
	}
	op.proxyPayer = a

	return nil
}
