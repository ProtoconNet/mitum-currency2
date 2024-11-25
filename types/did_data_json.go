package types

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"net/url"
)

type DataJSONMarshaler struct {
	hint.BaseHinter
	Address string `json:"address"`
	DID     string `json:"did"`
}

func (d Data) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(DataJSONMarshaler{
		BaseHinter: d.BaseHinter,
		Address:    d.address.String(),
		DID:        d.did.DID(),
	})
}

type DataJSONUnmarshaler struct {
	Hint    hint.Hint `json:"_hint"`
	Address string    `json:"address"`
	DID     string    `json:"did"`
}

func (d *Data) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of Data")

	var u DataJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	method, methodSpecificID, err := ParseDIDScheme(u.DID)
	if err != nil {
		return e.Wrap(err)
	}
	did := NewDIDResource(method, methodSpecificID)

	return d.unpack(enc, u.Hint, u.Address, did)
}

type DIDResourceJSONMarshaler struct {
	hint.BaseHinter
	Resource url.URL `json:"resource"`
}

func (d DIDResource) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(DIDResourceJSONMarshaler{
		BaseHinter: d.BaseHinter,
		Resource:   d.uriScheme,
	})
}

type DIDResourceJSONUnmarshaler struct {
	Hint     hint.Hint       `json:"_hint"`
	Resource json.RawMessage `json:"resource"`
}

func (d *DIDResource) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of DIDResource")

	var u DIDResourceJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	return d.unpack(enc, u.Hint, u.Resource)
}
