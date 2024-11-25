package types

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"go.mongodb.org/mongo-driver/bson"
)

type IAuthentication interface {
	ID() string
	AuthType() string
	Controller() string
	Details() interface{}
	bson.Marshaler
	util.Byter
	util.IsValider
}

var AsymmetricKeyAuthenticationHint = hint.MustNewHint("mitum-did-asymmetric-key-authentication-v0.0.1")
var SocialLogInAuthenticationHint = hint.MustNewHint("mitum-did-social-login-authentication-v0.0.1")
var VerificationMethodHint = hint.MustNewHint("mitum-did-verification-method-authentication-v0.0.1")

const AuthTypeED25519 = "Ed25519VerificationKey2020"
const AuthTypeECDSASECP = "EcdsaSecp256k1VerificationKey2019"
const AuthTypeVC = "VerifiableCredential"

type AsymmetricKeyAuthentication struct {
	hint.BaseHinter
	id         string
	authType   string
	controller string
	publicKey  base.Publickey
}

func NewAsymmetricKeyAuthentication(
	id, authType, controller string, publicKey base.Publickey,
) AsymmetricKeyAuthentication {
	return AsymmetricKeyAuthentication{
		BaseHinter: hint.NewBaseHinter(AsymmetricKeyAuthenticationHint),
		id:         id,
		authType:   authType,
		controller: controller,
		publicKey:  publicKey,
	}
}

func (d AsymmetricKeyAuthentication) IsValid([]byte) error {
	return nil
}

func (d AsymmetricKeyAuthentication) ID() string {
	return d.id
}

func (d AsymmetricKeyAuthentication) AuthType() string {
	return d.authType
}

func (d AsymmetricKeyAuthentication) Controller() string {
	return d.controller
}

func (d AsymmetricKeyAuthentication) Details() interface{} {
	return d.publicKey
}

func (d AsymmetricKeyAuthentication) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.id),
		[]byte(d.authType),
		[]byte(d.controller),
		d.publicKey.Bytes(),
	)
}

type SocialLogInAuthentication struct {
	hint.BaseHinter
	id              string
	authType        string
	controller      string
	serviceEndpoint string
	proof           Proof
}

func NewSocialLogInAuthentication(
	id, controller, serviceEndpoint string, proof Proof,
) SocialLogInAuthentication {
	return SocialLogInAuthentication{
		BaseHinter:      hint.NewBaseHinter(SocialLogInAuthenticationHint),
		id:              id,
		authType:        AuthTypeVC,
		controller:      controller,
		serviceEndpoint: serviceEndpoint,
		proof:           proof,
	}
}

func (d SocialLogInAuthentication) IsValid([]byte) error {
	return nil
}

func (d SocialLogInAuthentication) ID() string {
	return d.id
}

func (d SocialLogInAuthentication) AuthType() string {
	return d.authType
}

func (d SocialLogInAuthentication) Controller() string {
	return d.controller
}

func (d SocialLogInAuthentication) Details() interface{} {
	return map[string]interface{}{
		"serviceEndpoint": d.serviceEndpoint,
		"proof":           d.proof,
	}
}

func (d SocialLogInAuthentication) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.id),
		[]byte(d.authType),
		[]byte(d.controller),
		[]byte(d.serviceEndpoint),
		d.proof.Bytes(),
	)
}

type Proof struct {
	verificationMethod string
}

func NewProof(
	verificationMethod string,
) Proof {
	return Proof{
		verificationMethod: verificationMethod,
	}
}

func (d Proof) IsValid([]byte) error {
	return nil
}

func (d Proof) VerificationMethod() string {
	return d.verificationMethod
}

func (d Proof) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.verificationMethod),
	)
}

type IVerificationMethod interface {
	ID() string
	VerificationType() string
	Controller() string
	bson.Marshaler
	util.Byter
	util.IsValider
}

type VerificationMethod struct {
	hint.BaseHinter
	id               string
	verificationType string
	controller       string
	publicKey        string
}

func NewVerificationMethod(
	id, verificationType, controller, publicKey string,
) VerificationMethod {
	return VerificationMethod{
		BaseHinter:       hint.NewBaseHinter(VerificationMethodHint),
		id:               id,
		verificationType: verificationType,
		controller:       controller,
		publicKey:        publicKey,
	}
}

func (d VerificationMethod) ID() string {
	return d.id
}

func (d VerificationMethod) VerificationType() string {
	return d.verificationType
}

func (d VerificationMethod) Controller() string {
	return d.controller
}

func (d VerificationMethod) IsValid([]byte) error {
	return nil
}

func (d VerificationMethod) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(d.id),
		[]byte(d.verificationType),
		[]byte(d.controller),
		[]byte(d.publicKey),
	)
}
