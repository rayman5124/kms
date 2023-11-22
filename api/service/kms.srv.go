package srv

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	"kms/tutorial/cache"
	"kms/tutorial/common/utils/errutil"
)

type ans1PubKeyInfoFormat struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.ObjectIdentifier
}

type asn1PubKeyFormat struct {
	PublicKeyInfo ans1PubKeyInfoFormat
	PublicKey     asn1.BitString
}

type asn1SigFormat struct {
	R asn1.RawValue
	S asn1.RawValue
}

type asn1PKFormat struct {
	Version       int
	PrivateKey    []byte
	NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
	PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
}

type pkcs8Asn1PKFormat struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
}

type KmsSrv struct {
	client      *kms.Client
	pubKeyCache *cache.PubKeyCache
}

func NewKmsSrv(accessKey, secreteKey, region string) *KmsSrv {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secreteKey, "")
	cfg := errutil.HandleFatal(
		config.LoadDefaultConfig(
			context.Background(),
			config.WithCredentialsProvider(creds),
			config.WithRegion(region),
		),
	)

	return &KmsSrv{
		client:      kms.NewFromConfig(cfg),
		pubKeyCache: cache.NewPubKeyCache(),
	}
}

// 새로운 계정 생성
func (s *KmsSrv) CreateAccount() (*res.AccountRes, *errutil.ErrWrap) {
	key, err := s.client.CreateKey(context.TODO(), &kms.CreateKeyInput{
		KeyUsage: types.KeyUsageTypeSignVerify,
		KeySpec:  types.KeySpecEccSecgP256k1,
	})
	if err != nil {
		_, filteredErr := errutil.FilterAwsErr(err)
		return nil, errutil.NewErrWrap(500, "CreateAccount_kms_createKey", filteredErr)
	}
	keyID := *key.KeyMetadata.KeyId

	accountRes, errWrap := s.GetAccount(&dto.AccountDTO{KeyID: keyID})
	if errWrap != nil {
		errWrap.Code = 500
		return nil, errWrap
	}

	return accountRes, nil
}

// aws kms에 저장된 키들의 ID 리스트를 리턴
func (s *KmsSrv) GetAccountList(limit *int32, marker *string) (*res.AccountListRes, *errutil.ErrWrap) {
	keyList, err := s.client.ListKeys(context.TODO(), &kms.ListKeysInput{Limit: limit, Marker: marker})
	if err != nil {
		code, filteredErr := errutil.FilterAwsErr(err)
		return nil, errutil.NewErrWrap(code, "GetKeyIdList_kms_listekeys", filteredErr)
	}

	accountsList := make([]res.AccountRes, len(keyList.Keys))
	for i, key := range keyList.Keys {
		if key.KeyId != nil {
			// 삭제중인 키의 address를 가져올때 에러가 발생함으로 미리 필터링
			keyInfo, err := s.client.DescribeKey(context.TODO(), &kms.DescribeKeyInput{KeyId: key.KeyId})
			if err != nil {
				_, filteredErr := errutil.FilterAwsErr(err)
				return nil, errutil.NewErrWrap(500, "GetAccountList_kms_describeKey", filteredErr)
			}

			if keyInfo.KeyMetadata.Enabled {
				accountRes, errWrap := s.GetAccount(&dto.AccountDTO{KeyID: *key.KeyId})
				if errWrap != nil {
					errWrap.Code = 500
					return nil, errWrap
				}
				accountsList[i] = *accountRes
			} else {
				// 삭제중인 키는 address가 빈값
				accountsList[i] = res.AccountRes{KeyID: *key.KeyId}
			}
		}
	}

	if keyList.NextMarker != nil {
		return &res.AccountListRes{Accounts: accountsList, Marker: *keyList.NextMarker}, nil
	}

	return &res.AccountListRes{Accounts: accountsList}, nil
}

// keyID와 매칭되는 public key(버퍼)를 리턴
func (s *KmsSrv) GetPubkey(accountDTO *dto.AccountDTO) ([]byte, *errutil.ErrWrap) {
	pubkey, errRes := s.getPubKey(accountDTO.KeyID)
	if errRes != nil {
		return nil, errRes
	}
	return secp256k1.S256().Marshal(pubkey.X, pubkey.Y), nil
}

// keyID와 매칭되는 account를 리턴
func (s *KmsSrv) GetAccount(accountDTO *dto.AccountDTO) (*res.AccountRes, *errutil.ErrWrap) {
	pubkey, errWrap := s.getPubKey(accountDTO.KeyID)
	if errWrap != nil {
		return nil, errWrap
	}

	addr := crypto.PubkeyToAddress(*pubkey)
	return &res.AccountRes{KeyID: accountDTO.KeyID, Address: addr.String()}, nil
}

// 메세지에 서명 이후 R, S 값을 리턴
func (s *KmsSrv) Sign(keyID string, msg []byte) ([]byte, []byte, *errutil.ErrWrap) {
	signRes, err := s.client.Sign(context.TODO(), &kms.SignInput{
		KeyId:            aws.String(keyID),
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
		MessageType:      types.MessageTypeDigest,
		Message:          msg,
	})
	if err != nil {
		code, filteredErr := errutil.FilterAwsErr(err)
		return nil, nil, errutil.NewErrWrap(code, "Sign_kms_sign", filteredErr)
	}

	var sigAsn1 asn1SigFormat
	_, err = asn1.Unmarshal(signRes.Signature, &sigAsn1)
	if err != nil {
		return nil, nil, errutil.NewErrWrap(500, "Sign_asn1_unmarshal", err)
	}

	return sigAsn1.R.Bytes, sigAsn1.S.Bytes, nil

}

func (s *KmsSrv) getPubKey(keyID string) (*ecdsa.PublicKey, *errutil.ErrWrap) {
	cached := s.pubKeyCache.Get(keyID)
	if cached != nil {
		return cached, nil

	} else {
		pubKeyOut, err := s.client.GetPublicKey(context.TODO(), &kms.GetPublicKeyInput{
			KeyId: aws.String(keyID),
		})
		if err != nil {
			code, filteredErr := errutil.FilterAwsErr(err)
			return nil, errutil.NewErrWrap(code, "getPubKey_kms_getPublicKey", filteredErr)
		}

		var asn1PubKey asn1PubKeyFormat
		_, err = asn1.Unmarshal(pubKeyOut.PublicKey, &asn1PubKey)
		if err != nil {
			return nil, errutil.NewErrWrap(500, "getPubKey_asn1_unmarsahl", err)
		}

		pubKey, err := crypto.UnmarshalPubkey(asn1PubKey.PublicKey.Bytes)
		if err != nil {
			return nil, errutil.NewErrWrap(500, "getPubKey_crypto_unmarshalPubkey", err)
		}
		s.pubKeyCache.Add(keyID, pubKey)
		return pubKey, nil
	}
}

// 외부 private key를 주입
func (s *KmsSrv) ImportAccount(pk string) (*res.AccountRes, *errutil.ErrWrap) {
	// 특정 key-id 에 외부 pk를 주입한 이후 주입된 pk 를 삭제하고 다른 pk를 주입하는건 불가능하다
	// 한번이라도 외부키가 주입된 key-id는 이후로 계속 같은 외부키만 주입받을 수 있다.

	// kms key 껍데기 생성
	key, err := s.client.CreateKey(context.TODO(), &kms.CreateKeyInput{
		KeyUsage: types.KeyUsageTypeSignVerify,
		KeySpec:  types.KeySpecEccSecgP256k1,
		Origin:   types.OriginTypeExternal,
	})
	if err != nil {
		_, filteredErr := errutil.FilterAwsErr(err)
		return nil, errutil.NewErrWrap(500, "ImportAccount_kms_createKey", filteredErr)
	}
	keyID := *key.KeyMetadata.KeyId

	// private key 주입과정에서 필요한 파라미터값 요청
	importParameter, err := s.client.GetParametersForImport(context.TODO(), &kms.GetParametersForImportInput{
		KeyId:             &keyID,
		WrappingAlgorithm: types.AlgorithmSpecRsaesOaepSha256,
		WrappingKeySpec:   types.WrappingKeySpecRsa2048,
	})
	if err != nil {
		_, filteredErr := errutil.FilterAwsErr(err)
		return nil, errutil.NewErrWrap(500, "ImportAccount_kms_getParametersForImport", filteredErr)
	}

	// ==== private key ASN.1 데이터 형식으로 DER 인코딩 ====
	parsedPK, err := crypto.HexToECDSA(pk)
	if err != nil {
		return nil, errutil.NewErrWrap(500, "ImportAccount_crypto_hexToECDSA", err)
	}

	asn1EcPK, err := asn1.Marshal(asn1PKFormat{
		Version:       1,
		PrivateKey:    crypto.FromECDSA(parsedPK),
		PublicKey:     asn1.BitString{Bytes: crypto.FromECDSAPub(&parsedPK.PublicKey)},
		NamedCurveOID: asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1},
	})
	if err != nil {
		return nil, errutil.NewErrWrap(500, "ImportAccount_asn1_marshal", err)
	}

	pkcs8Asn1EcPK, err := asn1.Marshal(pkcs8Asn1PKFormat{
		Version: 0,
		Algo: pkix.AlgorithmIdentifier{
			Algorithm:  asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1},
			Parameters: asn1.RawValue{Class: 0, Tag: 6, IsCompound: false, Bytes: []uint8{0x2b, 0x81, 0x4, 0x0, 0xa}, FullBytes: []uint8{0x6, 0x5, 0x2b, 0x81, 0x4, 0x0, 0xa}},
		},
		PrivateKey: asn1EcPK,
	})
	if err != nil {
		return nil, errutil.NewErrWrap(500, "ImportAccount_asn1_marshal", err)
	}
	// ================================================

	parsedPubKey, err := x509.ParsePKIXPublicKey(importParameter.PublicKey)
	if err != nil {
		return nil, errutil.NewErrWrap(500, "ImportAccount_x509_parsePKIXPublicKey", err)
	}

	encryptedMaterial, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, parsedPubKey.(*rsa.PublicKey), pkcs8Asn1EcPK, nil)
	if err != nil {
		return nil, errutil.NewErrWrap(500, "ImportAccount_rsa_encryptOAEP", err)
	}

	_, err = s.client.ImportKeyMaterial(context.TODO(), &kms.ImportKeyMaterialInput{
		ImportToken:          importParameter.ImportToken,
		KeyId:                &keyID,
		EncryptedKeyMaterial: encryptedMaterial,
		ExpirationModel:      types.ExpirationModelTypeKeyMaterialDoesNotExpire,
	})
	if err != nil {
		_, filteredErr := errutil.FilterAwsErr(err)
		return nil, errutil.NewErrWrap(500, "ImportAccount_kms_importKeyMaterial", filteredErr)
	}

	accountRes, errWrap := s.GetAccount(&dto.AccountDTO{KeyID: keyID})
	if errWrap != nil {
		errWrap.Code = 500
		return nil, errWrap
	}
	return accountRes, nil
}
