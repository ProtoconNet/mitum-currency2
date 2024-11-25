package types

import (
	"github.com/ProtoconNet/mitum2/util/encoder"
)

func (d *DIDDocument) unpack(
	enc encoder.Encoder, context, id, created, status string, bSvc []byte,
) error {
	d.context_ = context
	d.id = id
	d.created = created
	d.status = status

	var svc Service
	err := enc.Unmarshal(bSvc, &svc)
	if err != nil {
		return err
	}
	d.service = svc

	return nil
}

func (d *Service) unpack(id, svcType, svcEP string) error {
	d.id = id
	d.serviceType = svcType
	d.serviceEndPoint = svcEP

	return nil
}

func (d *Proof) unpack(vrfM string) error {
	d.verificationMethod = vrfM

	return nil
}
