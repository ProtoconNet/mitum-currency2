package types

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

type DIDDocumentJSONMarshaler struct {
	hint.BaseHinter
	Context_  string                `json:"@context"`
	ID        string                `json:"id"`
	Created   string                `json:"created"`
	Status    string                `json:"status"`
	Auth      []IAuthentication     `json:"authentication"`
	VRFMethod []IVerificationMethod `json:"verificationMethod"`
	Service   Service               `json:"service"`
}

func (d DIDDocument) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(DIDDocumentJSONMarshaler{
		BaseHinter: d.BaseHinter,
		Context_:   d.context_,
		ID:         d.id,
		Created:    d.created,
		Status:     d.status,
		Auth:       d.authentication,
		VRFMethod:  d.verificationMethod,
		Service:    d.service,
	})
}

type DIDDocumentJSONUnmarshaler struct {
	Hint      hint.Hint       `json:"_hint"`
	Context_  string          `json:"@context"`
	ID        string          `json:"id"`
	Created   string          `json:"created"`
	Status    string          `json:"status"`
	Auth      json.RawMessage `json:"authentication"`
	VRFMethod json.RawMessage `json:"verificationMethod"`
	Service   json.RawMessage `json:"service"`
}

func (d *DIDDocument) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of %T", DIDDocument{})

	var u DIDDocumentJSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	d.BaseHinter = hint.NewBaseHinter(u.Hint)

	hr, err := enc.DecodeSlice(u.Auth)
	if err != nil {
		return err
	}

	auths := make([]IAuthentication, len(hr))
	for i, hinter := range hr {
		if v, ok := hinter.(IAuthentication); !ok {
			return e.Wrap(errors.Errorf("expected IAuthentication, not %T", hinter))
		} else {
			switch v.(type) {
			case AsymmetricKeyAuthentication:
				auth := v.(AsymmetricKeyAuthentication)
				if err := auth.IsValid(nil); err != nil {
					return e.Wrap(err)
				} else {
					auths[i] = auth
				}
			case SocialLogInAuthentication:
				auth := v.(SocialLogInAuthentication)
				if err := auth.IsValid(nil); err != nil {
					return e.Wrap(err)
				} else {
					auths[i] = auth
				}
			default:
			}
		}

	}
	d.authentication = auths

	hr, err = enc.DecodeSlice(u.VRFMethod)
	if err != nil {
		return err
	}

	vrfs := make([]IVerificationMethod, len(hr))
	for i, hinter := range hr {
		if v, ok := hinter.(IVerificationMethod); !ok {
			return e.Wrap(errors.Errorf("expected IVerificationMethod, not %T", hinter))
		} else {
			switch v.(type) {
			case VerificationMethod:
				auth := v.(VerificationMethod)
				if err := auth.IsValid(nil); err != nil {
					return e.Wrap(err)
				} else {
					vrfs[i] = auth
				}
			default:
			}
		}

	}
	d.verificationMethod = vrfs
	err = d.unpack(enc, u.Context_, u.ID, u.Created, u.Status, u.Service)
	if err != nil {
		return err
	}

	return nil
}

type ServiceJSONMarshaler struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndPoint string `json:"service_end_point"`
}

func (d Service) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(ServiceJSONMarshaler{
		ID:              d.id,
		Type:            d.serviceType,
		ServiceEndPoint: d.serviceEndPoint,
	})
}

type ServiceJSONUnmarshaler struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndPoint string `json:"service_end_point"`
}

func (d *Service) UnmarshalJSON(b []byte) error {
	e := util.StringError("failed to decode json of Service")

	var u ServiceJSONUnmarshaler
	if err := json.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.ID, u.Type, u.ServiceEndPoint)
}

type ProofJSONMarshaler struct {
	VerificationMethod string `json:"verificationMethod"`
}

func (d Proof) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(ProofJSONMarshaler{
		VerificationMethod: d.verificationMethod,
	})
}

type ProofJSONUnmarshaler struct {
	VerificationMethod string `json:"verificationMethod"`
}

func (d *Proof) UnmarshalJSON(b []byte) error {
	e := util.StringError("failed to decode json of Proof")

	var u ProofJSONUnmarshaler
	if err := json.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	err := d.unpack(u.VerificationMethod)
	if err != nil {
		return e.Wrap(err)
	}

	return nil
}
