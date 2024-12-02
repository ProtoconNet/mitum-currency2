package common

import (
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type BaseFactBSONUnmarshaler struct {
	Hash  string `bson:"hash"`
	Token []byte `bson:"token"`
}

type BaseSignBSONUnmarshaler struct {
	Signer    string         `bson:"signer"`
	Signature base.Signature `bson:"signature"`
	SignedAt  time.Time      `bson:"signed_at"`
}

type BaseOperationBSONUnmarshaler struct {
	Authentication bson.Raw `bson:"authentication"`
	Settlement     bson.Raw `bson:"settlement"`
}

func (op BaseOperation) MarshalBSON() ([]byte, error) {
	var signs bson.A

	for i := range op.signs {
		signs = append(signs, bson.M{
			"signer":    op.signs[i].Signer().String(),
			"signature": op.signs[i].Signature().String(),
			"signed_at": op.signs[i].SignedAt(),
		})
	}

	return bsonenc.Marshal(
		bson.M{
			"_hint":          op.Hint().String(),
			"hash":           op.Hash().String(),
			"fact":           op.Fact(),
			"signs":          signs,
			"authentication": op.Authentication,
			"settlement":     op.Settlement,
		})
}

func (op *BaseOperation) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var mbo MBaseOperation
	err := mbo.DecodeBSON(b, enc)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	op.MBaseOperation = mbo
	var u BaseOperationBSONUnmarshaler
	err = enc.Unmarshal(b, &u)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	var ba BaseAuthentication
	err = ba.DecodeBSON(u.Authentication, enc)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}
	op.SetAuthentication(ba)

	var bs BaseSettlement
	err = bs.DecodeBSON(u.Settlement, enc)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}
	op.SetSettlement(bs)

	return nil
}

type MBaseOperationBSONUnmarshaler struct {
	Hint  string     `bson:"_hint"`
	Hash  string     `bson:"hash"`
	Fact  bson.Raw   `bson:"fact"`
	Signs []bson.Raw `bson:"signs"`
}

func (op MBaseOperation) MarshalBSON() ([]byte, error) {
	var signs bson.A

	for i := range op.signs {
		signs = append(signs, bson.M{
			"signer":    op.signs[i].Signer().String(),
			"signature": op.signs[i].Signature().String(),
			"signed_at": op.signs[i].SignedAt(),
		})
	}

	return bsonenc.Marshal(
		bson.M{
			"_hint": op.Hint().String(),
			"hash":  op.Hash().String(),
			"fact":  op.Fact(),
			"signs": signs,
		},
	)
}

func (op *MBaseOperation) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var u MBaseOperationBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	op.BaseHinter = hint.NewBaseHinter(ht)
	op.h = valuehash.NewBytesFromString(u.Hash)

	var fact base.Fact
	if err := encoder.Decode(enc, u.Fact, &fact); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	op.SetFact(fact)

	var signs []base.Sign

	for i := range u.Signs {
		var us BaseSignBSONUnmarshaler
		var pubKey base.Publickey
		var err error
		if err = enc.Unmarshal(u.Signs[i], &us); err != nil {
			return DecorateError(err, ErrDecodeBson, *op)
		}

		if pubKey, err = base.DecodePublickeyFromString(us.Signer, enc); err != nil {
			return DecorateError(err, ErrDecodeBson, *op)
		}

		sign := base.NewBaseSign(pubKey, us.Signature, us.SignedAt)
		signs = append(signs, sign)
	}
	op.signs = signs

	return nil
}

func (op *BaseNodeOperation) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var u MBaseOperationBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	op.MBaseOperation.BaseHinter = hint.NewBaseHinter(ht)
	op.MBaseOperation.h = valuehash.NewBytesFromString(u.Hash)

	var fact base.Fact
	if err := encoder.Decode(enc, u.Fact, &fact); err != nil {
		return DecorateError(err, ErrDecodeBson, *op)
	}

	op.MBaseOperation.SetFact(fact)

	return nil
}
