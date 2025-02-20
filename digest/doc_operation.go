package digest

import (
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/pkg/errors"
	"time"

	mongodbst "github.com/ProtoconNet/mitum-currency/v3/digest/mongodb"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type OperationDoc struct {
	mongodbst.BaseDoc
	va        OperationValue
	op        base.Operation
	addresses []string
	height    base.Height
}

func NewOperationDoc(
	op base.Operation,
	enc encoder.Encoder,
	height base.Height,
	confirmedAt time.Time,
	inState bool,
	reason string,
	index uint64,
) (OperationDoc, error) {
	var addresses []string
	if ads, ok := op.Fact().(types.Addresses); ok {
		as, err := ads.Addresses()
		if err != nil {
			return OperationDoc{}, err
		}
		addresses = make([]string, len(as))
		for i := range as {
			addresses[i] = as[i].String()
		}
	}
	if opExt, ok := op.(extras.OperationExtensions); ok {
		iSettlement := opExt.Extension(extras.SettlementExtensionType)
		iProxyPayer := opExt.Extension(extras.ProxyPayerExtensionType)
		if iSettlement != nil {
			settlement, ok := iSettlement.(extras.Settlement)
			if !ok {
				return OperationDoc{}, errors.Errorf("expected Settlement, but %T", iSettlement)
			}
			opSender := settlement.OpSender()
			if opSender != nil {
				addresses = append(addresses, opSender.String())
			}
		}
		if iProxyPayer != nil {
			proxyPayer, ok := iProxyPayer.(extras.ProxyPayer)
			if !ok {
				return OperationDoc{}, errors.Errorf("expected ProxyPayer, but %T", iProxyPayer)
			}
			if proxyPayer := proxyPayer.ProxyPayer(); proxyPayer != nil {
				addresses = append(addresses, proxyPayer.String())
			}
		}
	}

	va := NewOperationValue(op, height, confirmedAt, inState, reason, index)
	b, err := mongodbst.NewBaseDoc(nil, va, enc)
	if err != nil {
		return OperationDoc{}, err
	}

	return OperationDoc{
		BaseDoc:   b,
		va:        va,
		op:        op,
		addresses: addresses,
		height:    height,
	}, nil
}

func (doc OperationDoc) MarshalBSON() ([]byte, error) {
	m, err := doc.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["addresses"] = doc.addresses
	m["fact"] = doc.op.Fact().Hash()
	m["height"] = doc.height
	m["index"] = doc.va.index

	return bsonenc.Marshal(m)
}
