package types

import (
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

var DIDDocumentHint = hint.MustNewHint("mitum-did-document-v0.0.1")

type DIDDocument struct {
	hint.BaseHinter
	context_           string
	id                 string
	authentication     []IAuthentication
	verificationMethod []IVerificationMethod
	service            Service
}

func NewDIDDocument(
	did string, auth []IAuthentication, vrf []IVerificationMethod, service Service,
) DIDDocument {
	return DIDDocument{
		BaseHinter:         hint.NewBaseHinter(DIDDocumentHint),
		context_:           "https://www.w3.org/ns/did/v1",
		id:                 did,
		authentication:     auth,
		verificationMethod: vrf,
		service:            service,
	}
}

func (d DIDDocument) IsValid([]byte) error {
	foundMap := map[string]struct{}{}
	for _, v := range d.authentication {
		if _, found := foundMap[v.ID()]; found {
			return errors.Errorf("duplicated authentication id found")
		}
		foundMap[v.ID()] = struct{}{}
	}

	foundMap = map[string]struct{}{}
	for _, v := range d.verificationMethod {
		if _, found := foundMap[v.ID()]; found {
			return errors.Errorf("duplicated verificationMethod id found")
		}
		foundMap[v.ID()] = struct{}{}
	}
	return nil
}

func (d DIDDocument) Bytes() []byte {
	var bAuth [][]byte
	for _, v := range d.authentication {
		bAuth = append(bAuth, v.Bytes())
	}
	byteAuth := util.ConcatBytesSlice(bAuth...)

	var bVrf [][]byte
	for _, v := range d.verificationMethod {
		bVrf = append(bVrf, v.Bytes())
	}
	byteVrf := util.ConcatBytesSlice(bVrf...)

	return util.ConcatBytesSlice(
		[]byte(d.context_),
		[]byte(d.id),
		byteAuth,
		byteVrf,
		d.service.Bytes(),
	)
}

func (d DIDDocument) DID() string {
	return d.id
}

func (d DIDDocument) Authentication(id string) (IAuthentication, error) {
	for _, v := range d.authentication {
		if v.ID() == id {
			return v, nil
		}
	}

	return nil, errors.Errorf("Authentication not found by id %v", id)
}

func (d DIDDocument) VerificationMethod(id string) (IVerificationMethod, error) {
	for _, v := range d.verificationMethod {
		if v.ID() == id {
			return v, nil
		}
	}

	return nil, errors.Errorf("VerificationMethod not found by id %v", id)
}

type Service struct {
	id              string
	serviceType     string
	serviceEndPoint string
}

func NewService(
	id, serviceType, serviceEndPoint string,
) Service {
	return Service{
		id:              id,
		serviceType:     serviceType,
		serviceEndPoint: serviceEndPoint,
	}
}

func (d Service) IsValid([]byte) error {
	return nil
}

func (d Service) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.id),
		[]byte(d.serviceType),
		[]byte(d.serviceEndPoint),
	)
}
