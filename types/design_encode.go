package types

import (
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (de *Design) unmarshal(
	_ encoder.Encoder,
	ht hint.Hint,
	didMethod string,
) error {
	de.BaseHinter = hint.NewBaseHinter(ht)
	de.didMethod = didMethod

	return nil
}
