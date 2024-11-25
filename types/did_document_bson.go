package types

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (d DIDDocument) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint":              d.Hint().String(),
		"@context":           d.context_,
		"id":                 d.id,
		"created":            d.created,
		"status":             d.status,
		"authentication":     d.authentication,
		"verificationMethod": d.verificationMethod,
		"service":            d.service,
	})
}

type DIDDocumentBSONUnmarshaler struct {
	Hint      string   `bson:"_hint"`
	Context_  string   `bson:"@context"`
	ID        string   `bson:"id"`
	Created   string   `bson:"created"`
	Status    string   `bson:"status"`
	Auth      bson.Raw `bson:"authentication"`
	VRFMethod bson.Raw `bson:"verificationMethod"`
	Service   bson.Raw `bson:"service"`
}

func (d *DIDDocument) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("decode bson of DIDDocument")

	var u DIDDocumentBSONUnmarshaler
	if err := bsonenc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	d.BaseHinter = hint.NewBaseHinter(ht)

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

	return d.unpack(enc, u.Context_, u.ID, u.Created, u.Status, u.Service)
}

func (d Service) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"id":                d.id,
		"type":              d.serviceType,
		"service_end_point": d.serviceEndPoint,
	})
}

type ServiceBSONUnmarshaler struct {
	ID              string `bson:"id"`
	Type            string `bson:"type"`
	ServiceEndPoint string `bson:"service_end_point"`
}

func (d *Service) UnmarshalBSON(b []byte) error {
	e := util.StringError("decode bson of Service")

	var u ServiceBSONUnmarshaler
	err := bsonenc.Unmarshal(b, &u)
	if err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.ID, u.Type, u.ServiceEndPoint)
}

func (d Proof) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"verificationMethod": d.verificationMethod,
	})
}

type ProofBSONUnmarshaler struct {
	VerificationMethod string `bson:"verificationMethod"`
}

func (d *Proof) UnmarshalBSON(b []byte) error {
	e := util.StringError("decode bson of Proof")

	var u ProofBSONUnmarshaler
	err := bsonenc.Unmarshal(b, &u)
	if err != nil {
		return e.Wrap(err)
	}

	return d.unpack(u.VerificationMethod)
}
