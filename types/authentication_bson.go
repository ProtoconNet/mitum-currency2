package types

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/hint"
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
)

func (d AsymmetricKeyAuthentication) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint":        d.Hint().String(),
		"id":           d.id,
		"authType":     d.authType,
		"controller":   d.controller,
		"publicKeyHex": d.publicKey.String(),
	})
}

type AsymmetricKeyAuthenticationBSONUnmarshaler struct {
	Hint       string `bson:"_hint"`
	ID         string `bson:"id"`
	AuthType   string `bson:"authType"`
	Controller string `bson:"controller"`
	PublicKey  string `bson:"publicKeyHex"`
}

func (d *AsymmetricKeyAuthentication) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("decode bson of AsymmetricKeyAuthentication")

	var u AsymmetricKeyAuthenticationBSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	d.BaseHinter = hint.NewBaseHinter(ht)

	pubKey, err := base.DecodePublickeyFromString(u.PublicKey, enc)
	if err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.ID, u.AuthType, u.Controller, pubKey)
}

func (d SocialLogInAuthentication) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint":           d.Hint().String(),
		"id":              d.id,
		"authType":        d.authType,
		"controller":      d.controller,
		"serviceEndpoint": d.serviceEndpoint,
		"proof":           d.proof,
	})
}

type SocialLogInAuthenticationBSONUnmarshaler struct {
	Hint            string   `bson:"_hint"`
	ID              string   `bson:"id"`
	AuthType        string   `bson:"authType"`
	Controller      string   `bson:"controller"`
	ServiceEndPoint string   `bson:"serviceEndpoint"`
	Proof           bson.Raw `bson:"proof"`
}

func (d *SocialLogInAuthentication) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("decode bson of SocialLogInAuthentication")

	var u SocialLogInAuthenticationBSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	d.BaseHinter = hint.NewBaseHinter(ht)

	var p Proof
	err = enc.Unmarshal(u.Proof, &p)
	if err != nil {
		return e.Wrap(err)
	}
	d.proof = p

	return d.unpack(u.ID, u.AuthType, u.Controller, u.ServiceEndPoint)
}

func (d VerificationMethod) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint":            d.Hint().String(),
		"id":               d.id,
		"verificationType": d.verificationType,
		"controller":       d.controller,
		"publicKeyHex":     d.publicKey,
	})
}

type VerificationMethodBSONUnmarshaler struct {
	Hint             string `bson:"_hint"`
	ID               string `bson:"id"`
	VerificationType string `bson:"verificationType"`
	Controller       string `bson:"controller"`
	PublicKey        string `bson:"publicKeyHex"`
}

func (d *VerificationMethod) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("decode bson of SocialLogInAuthentication")

	var u VerificationMethodBSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	d.BaseHinter = hint.NewBaseHinter(ht)

	return d.unpack(u.ID, u.VerificationType, u.Controller, u.PublicKey)
}
