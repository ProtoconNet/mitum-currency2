package isaacoperation

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type suffrageDisjoinFactJSONMarshaler struct {
	Node base.Address `json:"node"`
	base.BaseFactJSONMarshaler
	Start base.Height `json:"start"`
}

func (fact SuffrageDisjoinFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(suffrageDisjoinFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Node:                  fact.node,
		Start:                 fact.start,
	})
}

type suffrageDisjoinFactJSONUnmarshaler struct {
	Node string `json:"node"`
	base.BaseFactJSONUnmarshaler
	Start base.Height `json:"start"`
}

func (fact *SuffrageDisjoinFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode SuffrageDisjoinFact")

	var u suffrageDisjoinFactJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	fact.BaseFact.SetJSONUnmarshaler(u.BaseFactJSONUnmarshaler)

	return fact.unpack(enc, u.Node, u.Start)
}
