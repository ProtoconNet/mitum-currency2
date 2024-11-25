package types

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"net/url"
)

func (d *Data) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	pubKey string, did DIDResource,
) error {
	d.BaseHinter = hint.NewBaseHinter(ht)
	a, err := base.DecodeAddress(pubKey, enc)
	if err != nil {
		return err
	}
	d.address = a

	d.did = did

	return nil
}

func (d *DIDResource) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	rsc []byte,
) error {
	d.BaseHinter = hint.NewBaseHinter(ht)

	var u url.URL
	err := enc.Unmarshal(rsc, &u)
	if err != nil {
		return err
	}

	d.uriScheme = u

	return nil
}
