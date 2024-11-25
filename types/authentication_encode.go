package types

import "github.com/ProtoconNet/mitum2/base"

func (d *AsymmetricKeyAuthentication) unpack(
	id, authType, controller string, publicKey base.Publickey,
) error {
	d.id = id
	d.authType = authType
	d.controller = controller

	d.publicKey = publicKey

	return nil
}

func (d *SocialLogInAuthentication) unpack(
	id, authType, controller, serviceEP string,
) error {
	d.id = id
	d.authType = authType
	d.controller = controller
	d.serviceEndpoint = serviceEP

	return nil
}

func (d *VerificationMethod) unpack(
	id, vrfType, controller, pubKey string,
) error {
	d.id = id
	d.verificationType = vrfType
	d.controller = controller
	d.publicKey = pubKey

	return nil
}
