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
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/cache"
	"kms/wallet/common/errs"
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

func NewKmsSrv(kmsClient *kms.Client) *KmsSrv {
	return &KmsSrv{kmsClient, cache.NewPubKeyCache()}
}

// 새로운 계정 생성
func (s *KmsSrv) CreateAccount() (*dto.AccountRes, error) {
	key, err := s.client.CreateKey(context.TODO(), &kms.CreateKeyInput{
		KeyUsage: types.KeyUsageTypeSignVerify,
		KeySpec:  types.KeySpecEccSecgP256k1,
	})
	if err != nil {
		return nil, errs.RouteAwsErr(err)

	}
	keyID := *key.KeyMetadata.KeyId

	accountRes, err := s.GetAccount(&dto.KeyIdReq{KeyID: keyID})
	if err != nil {
		return nil, err
	}

	return accountRes, nil
}

// keyID와 매칭되는 account 리턴
func (s *KmsSrv) GetAccount(keyIdDTO *dto.KeyIdReq) (*dto.AccountRes, error) {
	pubkey, err := s.getPubKey(keyIdDTO.KeyID)
	if err != nil {
		return nil, err
	}

	addr := crypto.PubkeyToAddress(*pubkey)
	return &dto.AccountRes{Address: addr.String(), KeyID: keyIdDTO.KeyID}, nil
}

// aws kms에 저장된 키들의 ID 리스트를 리턴
func (s *KmsSrv) GetAccountList(accountListDTO *dto.AccountListReq) (*dto.AccountListRes, error) {
	keyList, err := s.client.ListKeys(context.TODO(), &kms.ListKeysInput{
		Limit:  accountListDTO.Limit,
		Marker: accountListDTO.Marker,
	})
	if err != nil {
		return nil, errs.RouteAwsErr(err)
	}

	accountsList := make([]dto.AccountRes, len(keyList.Keys))
	for i, key := range keyList.Keys {
		if key.KeyId != nil {
			// 사용 불가능한 키는 필터링 한다
			keyInfo, err := s.client.DescribeKey(context.TODO(), &kms.DescribeKeyInput{KeyId: key.KeyId})
			if err != nil {
				return nil, errs.RouteAwsErr(err)
			}
			if keyInfo.KeyMetadata.Enabled && keyInfo.KeyMetadata.KeySpec == types.KeySpecEccSecgP256k1 {
				accountRes, err := s.GetAccount(&dto.KeyIdReq{KeyID: *key.KeyId})
				if err != nil {
					return nil, err
				}
				accountsList[i] = *accountRes
			} else {
				// 사용불가한 계정은 address 를 빈값으로 리턴한다
				accountsList[i] = dto.AccountRes{KeyID: *key.KeyId}
			}
		}
	}

	if keyList.NextMarker != nil {
		return &dto.AccountListRes{Accounts: accountsList, Marker: *keyList.NextMarker}, nil
	}

	return &dto.AccountListRes{Accounts: accountsList}, nil
}

// 외부 private key를 주입
func (s *KmsSrv) ImportAccount(pkDTO *dto.PkReq) (*dto.AccountRes, error) {
	// 특정 key-id 에 외부 pk를 주입한 이후 주입된 pk 를 삭제하고 다른 pk를 주입하는건 불가능하다
	// 한번이라도 외부키가 주입된 key-id는 이후로 계속 같은 외부키만 주입받을 수 있다.

	var (
		keyID           *string
		importParameter *kms.GetParametersForImportOutput
		errChan         = make(chan error)
	)

	go func() {
		// kms key 껍데기 생성
		key, err := s.client.CreateKey(context.TODO(), &kms.CreateKeyInput{
			KeyUsage: types.KeyUsageTypeSignVerify,
			KeySpec:  types.KeySpecEccSecgP256k1,
			Origin:   types.OriginTypeExternal,
		})
		if err != nil {
			errChan <- errs.RouteAwsErr(err)
			return
		}
		keyID = key.KeyMetadata.KeyId

		// private key 주입과정에서 필요한 파라미터값 요청
		importParameter, err = s.client.GetParametersForImport(context.TODO(), &kms.GetParametersForImportInput{
			KeyId:             keyID,
			WrappingAlgorithm: types.AlgorithmSpecRsaesOaepSha256,
			WrappingKeySpec:   types.WrappingKeySpecRsa2048,
		})
		if err != nil {
			errChan <- errs.RouteAwsErr(err)
			return
		}

		errChan <- nil
	}()

	// ==== private key ASN.1 데이터 형식으로 DER 인코딩 ====
	ecdsaPK, err := crypto.HexToECDSA(pkDTO.PK)
	if err != nil {
		return nil, errs.InternalServerErr(err)
	}

	asn1EcPK, err := asn1.Marshal(asn1PKFormat{
		Version:       1,
		PrivateKey:    crypto.FromECDSA(ecdsaPK),
		PublicKey:     asn1.BitString{Bytes: crypto.FromECDSAPub(&ecdsaPK.PublicKey)},
		NamedCurveOID: asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1},
	})
	if err != nil {
		return nil, errs.InternalServerErr(err)
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
		return nil, errs.InternalServerErr(err)
	}
	// ================================================

	if errWrap := <-errChan; errWrap != nil {
		return nil, errWrap
	}

	rsaPubKey, err := x509.ParsePKIXPublicKey(importParameter.PublicKey)
	if err != nil {
		return nil, errs.InternalServerErr(err)
	}
	encryptedMaterial, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey.(*rsa.PublicKey), pkcs8Asn1EcPK, nil)
	if err != nil {
		return nil, errs.InternalServerErr(err)
	}

	_, err = s.client.ImportKeyMaterial(context.TODO(), &kms.ImportKeyMaterialInput{
		ImportToken:          importParameter.ImportToken,
		KeyId:                keyID,
		EncryptedKeyMaterial: encryptedMaterial,
		ExpirationModel:      types.ExpirationModelTypeKeyMaterialDoesNotExpire,
	})
	if err != nil {
		return nil, errs.RouteAwsErr(err)
	}

	accountRes, errWrap := s.GetAccount(&dto.KeyIdReq{KeyID: *keyID})
	if errWrap != nil {
		return nil, errs.RouteAwsErr(err)
	}
	return accountRes, nil
}

func (s *KmsSrv) DeleteAccount(keyIdDTO *dto.KeyIdReq) (*dto.AccountDeletionRes, error) {
	output, err := s.client.ScheduleKeyDeletion(context.TODO(), &kms.ScheduleKeyDeletionInput{
		KeyId:               &keyIdDTO.KeyID,
		PendingWindowInDays: aws.Int32(7),
	})
	if err != nil {
		return nil, errs.RouteAwsErr(err)
	}

	return &dto.AccountDeletionRes{KeyID: keyIdDTO.KeyID, DeletionDate: output.DeletionDate.String()}, nil
}

// 메세지에 서명 이후 R, S 값을 리턴
func (s *KmsSrv) Sign(keyID string, msg []byte) ([]byte, []byte, error) {
	signRes, err := s.client.Sign(context.TODO(), &kms.SignInput{
		KeyId:            aws.String(keyID),
		SigningAlgorithm: types.SigningAlgorithmSpecEcdsaSha256,
		MessageType:      types.MessageTypeDigest,
		Message:          msg,
	})
	if err != nil {
		return nil, nil, errs.RouteAwsErr(err)
	}

	var sigAsn1 asn1SigFormat
	_, err = asn1.Unmarshal(signRes.Signature, &sigAsn1)
	if err != nil {
		return nil, nil, errs.InternalServerErr(err)
	}

	return sigAsn1.R.Bytes, sigAsn1.S.Bytes, nil
}

// keyID와 매칭되는 public key(바이트)를 리턴
func (s *KmsSrv) GetPubkey(keyIdDTO *dto.KeyIdReq) ([]byte, error) {
	pubkey, err := s.getPubKey(keyIdDTO.KeyID)
	if err != nil {
		return nil, err
	}
	return secp256k1.S256().Marshal(pubkey.X, pubkey.Y), nil
}

func (s *KmsSrv) getPubKey(keyID string) (*ecdsa.PublicKey, error) {
	cached := s.pubKeyCache.Get(keyID)
	if cached != nil {
		return cached, nil

	}
	pubKeyOut, err := s.client.GetPublicKey(context.TODO(), &kms.GetPublicKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, errs.RouteAwsErr(err)
	}

	var asn1PubKey asn1PubKeyFormat
	_, err = asn1.Unmarshal(pubKeyOut.PublicKey, &asn1PubKey)
	if err != nil {
		return nil, errs.InternalServerErr(err)
	}

	pubKey, err := crypto.UnmarshalPubkey(asn1PubKey.PublicKey.Bytes)
	if err != nil {
		return nil, errs.InternalServerErr(err)
	}
	s.pubKeyCache.Add(keyID, pubKey)
	return pubKey, nil

}
