package types

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

type AsymmetricKeyAuthenticationJSONMarshaler struct {
	hint.BaseHinter
	ID         string `json:"id"`
	AuthType   string `json:"authType"`
	Controller string `json:"controller"`
	PublicKey  string `json:"publicKey"`
}

func (d AsymmetricKeyAuthentication) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(AsymmetricKeyAuthenticationJSONMarshaler{
		BaseHinter: d.BaseHinter,
		ID:         d.id,
		AuthType:   d.authType,
		Controller: d.controller,
		PublicKey:  d.publicKey.String(),
	})
}

type AsymmetricKeyAuthenticationJSONUnmarshaler struct {
	Hint       hint.Hint `json:"_hint"`
	ID         string    `json:"id"`
	AuthType   string    `json:"authType"`
	Controller string    `json:"controller"`
	PublicKey  string    `json:"publicKey"`
}

func (d *AsymmetricKeyAuthentication) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of %T", AsymmetricKeyAuthentication{})

	var u AsymmetricKeyAuthenticationJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	pubKey, err := base.DecodePublickeyFromString(u.PublicKey, enc)
	if err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.ID, u.AuthType, u.Controller, pubKey)
}

type SocialLogInAuthenticationJSONMarshaler struct {
	hint.BaseHinter
	ID              string `json:"id"`
	AuthType        string `json:"authType"`
	Controller      string `json:"controller"`
	ServiceEndpoint string `json:"serviceEndpoint"`
	Proof           Proof  `json:"proof"`
}

func (d SocialLogInAuthentication) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(SocialLogInAuthenticationJSONMarshaler{
		BaseHinter:      d.BaseHinter,
		ID:              d.id,
		AuthType:        d.authType,
		Controller:      d.controller,
		ServiceEndpoint: d.serviceEndpoint,
		Proof:           d.proof,
	})
}

type SocialLogInAuthenticationJSONUnmarshaler struct {
	Hint            hint.Hint       `json:"_hint"`
	ID              string          `json:"id"`
	AuthType        string          `json:"authType"`
	Controller      string          `json:"controller"`
	ServiceEndpoint string          `json:"serviceEndpoint"`
	Proof           json.RawMessage `json:"proof"`
}

func (d *SocialLogInAuthentication) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of %T", SocialLogInAuthentication{})

	var u SocialLogInAuthenticationJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	var p Proof
	err := enc.Unmarshal(u.Proof, &p)
	if err != nil {
		return e.Wrap(err)
	}
	d.proof = p

	return d.unpack(u.ID, u.AuthType, u.Controller, u.ServiceEndpoint)
}

type VerificationMethodJSONMarshaler struct {
	hint.BaseHinter
	ID         string `json:"id"`
	VRFType    string `json:"verificationType"`
	Controller string `json:"controller"`
	PublicKey  string `json:"publicKeyHex"`
}

func (d VerificationMethod) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(VerificationMethodJSONMarshaler{
		BaseHinter: d.BaseHinter,
		ID:         d.id,
		VRFType:    d.verificationType,
		Controller: d.controller,
		PublicKey:  d.publicKey,
	})
}

type VerificationMethodJSONUnmarshaler struct {
	Hint       hint.Hint `json:"_hint"`
	ID         string    `json:"id"`
	VRFType    string    `json:"verificationType"`
	Controller string    `json:"controller"`
	PublicKey  string    `json:"publicKeyHex"`
}

func (d *VerificationMethod) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of %T", VerificationMethod{})

	var u VerificationMethodJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.ID, u.VRFType, u.Controller, u.PublicKey)
}

type ServiceMarshaler struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndPoint string `json:"serviceEndpoint"`
}
