package types

import (
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
	"net/url"
	"strings"
)

const MinKeyLen = 128
const DIDPrefix = "did"
const HexPrefix = "0x"
const AddressSuffix = "fca"
const DIDSeparator = ":"

var DataHint = hint.MustNewHint("mitum-did-data-v0.0.1")
var DIDResourceHint = hint.MustNewHint("mitum-did-resource-v0.0.1")
var DocumentHint = hint.MustNewHint("mitum-did-document-v0.0.1")

type Data struct {
	hint.BaseHinter
	address base.Address
	did     DIDResource
}

func NewData(
	address base.Address, method string,
) Data {
	data := Data{
		BaseHinter: hint.NewBaseHinter(DataHint),
	}
	data.address = address
	data.did = NewDIDResource(method, address.String())
	return data
}

func (d Data) IsValid([]byte) error {
	return nil
}

func (d Data) Bytes() []byte {
	return util.ConcatBytesSlice(
		d.address.Bytes(),
		d.did.Bytes(),
	)
}

func (d Data) Address() base.Address {
	return d.address
}

func (d Data) DID() string {
	return d.did.DID()
}

func (d Data) DIDResource() DIDResource {
	return d.did
}

func (d Data) Equal(b Data) bool {
	if d.address.Equal(b.address) {
		return false
	}
	if d.did.DID() != b.did.DID() {
		return false
	}

	return true
}

type DIDResource struct {
	hint.BaseHinter
	method           string
	methodSpecificID string
	uriScheme        url.URL
}

func NewDIDResource(method, methodSpecificID string) DIDResource {
	didResource := url.URL{
		Scheme: DIDPrefix,
		Opaque: fmt.Sprintf("%s%s%s", method, DIDSeparator, methodSpecificID),
	}

	return DIDResource{
		BaseHinter:       hint.NewBaseHinter(DIDResourceHint),
		method:           method,
		methodSpecificID: methodSpecificID,
		uriScheme:        didResource,
	}
}

func NewDIDResourceFromString(didURL string) (*DIDResource, error) {
	u, err := url.Parse(didURL)
	if err != nil {
		return nil, common.ErrValueInvalid.Wrap(err)
	}

	didResource := u
	did := fmt.Sprintf("%s:%s", DIDPrefix, didResource.Opaque)
	method, methodSpecificID, err := ParseDIDScheme(did)
	if err != nil {
		return nil, common.ErrValueInvalid.Wrap(err)
	}

	return &DIDResource{
		BaseHinter:       hint.NewBaseHinter(DIDResourceHint),
		method:           method,
		methodSpecificID: methodSpecificID,
		uriScheme:        *didResource,
	}, nil
}

func (d DIDResource) DID() string {
	return fmt.Sprintf("%s:%s", DIDPrefix, d.uriScheme.Opaque)
}

func (d DIDResource) Method() string {
	return d.method
}

func (d DIDResource) MethodSpecificID() string {
	return d.methodSpecificID
}

func (d DIDResource) DIDUrl() string {
	return d.uriScheme.String()
}

func (d *DIDResource) SetFragment(fragment string) {
	d.uriScheme.Fragment = fragment
}

func (d DIDResource) IsValid([]byte) error {
	didStrings := strings.Split(d.DID(), DIDSeparator)
	if len(didStrings) != 3 {
		return errors.Errorf("invalid DID scheme, %v", d.DID())
	}
	if didStrings[0] != DIDPrefix {
		return errors.Errorf("invalid DID scheme, %v", d.DID())
	}

	return nil
}

func (d DIDResource) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.uriScheme.Scheme),
		[]byte(d.uriScheme.Path),
		[]byte(d.uriScheme.Fragment),
	)
}

func (d DIDResource) Equal(b DIDResource) bool {
	if d.uriScheme.Scheme != b.uriScheme.Scheme {
		return false
	} else if d.uriScheme.Path != b.uriScheme.Path {
		return false
	} else if d.uriScheme.Fragment != b.uriScheme.Fragment {
		return false
	}

	return true
}

func ParseDIDScheme(did string) (method, methodSpecificID string, err error) {
	didStrings := strings.Split(did, DIDSeparator)
	if len(didStrings) != 3 {
		err = errors.Errorf("invalid DID scheme, %v", did)
		return
	}

	if didStrings[0] != DIDPrefix {
		err = errors.Errorf("invalid DID scheme, %v", did)
		return
	}

	method = didStrings[1]
	methodSpecificID = didStrings[2]
	return
}
