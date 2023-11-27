package srv

import (
	"encoding/hex"
	"fmt"
	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	"kms/tutorial/common/utils/errutil"
	"kms/tutorial/common/utils/ethutil"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type TxnSrv struct {
	chainID *big.Int
	kmsSrv  *KmsSrv
}

func NewTxnSrv(chainID *big.Int, kmsSrv *KmsSrv) *TxnSrv {
	return &TxnSrv{chainID, kmsSrv}
}

// 서명되지 않은 트렌젝션을 받아서, 서명한뒤 리턴
func (s *TxnSrv) SignSerializedTxn(txnDTO *dto.TxnDTO) (*res.SingedTxnRes, *errutil.ErrWrap) {
	parsedTxn, errWrap := s.parseTxn(txnDTO.SerializedTxn)
	if errWrap != nil {
		return nil, errWrap
	}

	signer := types.NewCancunSigner(s.chainID)
	txnMsg := signer.Hash(parsedTxn).Bytes()

	// kms로부터 서명을 받아온다
	R, S, errWrap := s.kmsSrv.Sign(txnDTO.KeyID, txnMsg)
	if errWrap != nil {
		return nil, errWrap
	}

	// S값 가공
	secp256k1n := crypto.S256().Params().N                        // secp256k1 타원곡선의 최댓값
	halfSecp256k1n := new(big.Int).Div(secp256k1n, big.NewInt(2)) // 타원곡선의 최대값의 절반
	// S 값이 타원곡선 최댓값의 절반보다 크면 변환해서 사용 (reference -> EIP2 https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md)
	if sBigInt := new(big.Int).SetBytes(S); sBigInt.Cmp(halfSecp256k1n) > 0 {
		S = new(big.Int).Sub(secp256k1n, sBigInt).Bytes()

		// 원래 ECDSA 서명 방식에서 기존 S, curve.n - S 둘다 유효한 값이지만 이더리움에서는 후자만 유효하다
	}

	pubKey, errWrap := s.kmsSrv.GetPubkey(&dto.AddressDTO{KeyID: txnDTO.KeyID})
	if errWrap != nil {
		return nil, errWrap
	}

	// V 값을 유추해서 완전한 이더리움 서명을 만든다
	signature, errWrap := s.getFullSignature(txnMsg, R, S, pubKey)
	if errWrap != nil {
		return nil, errWrap
	}

	signedTxn, err := parsedTxn.WithSignature(signer, signature)
	if err != nil {
		return nil, errutil.NewErrWrap(500, "SignSerializedTxn_types.transction_withSignature", err)
	}

	byteSignedTxn, err := signedTxn.MarshalBinary()
	if err != nil {
		return nil, errutil.NewErrWrap(500, "SignSerializedTxn_types.transaction_marshalBinary", err)
	}
	return &res.SingedTxnRes{SignedTxn: "0x" + common.Bytes2Hex(byteSignedTxn)}, nil
}

// 직렬화된 트렌젝션 데이터를 type.Transaction Struct로 변환
func (s *TxnSrv) parseTxn(serializedTxn string) (*types.Transaction, *errutil.ErrWrap) {
	var parsedTxn types.Transaction

	err := parsedTxn.UnmarshalBinary(common.FromHex(serializedTxn))
	if err != nil {
		return nil, errutil.NewErrWrap(400, "", err)
	}

	return &parsedTxn, nil
}

// r, s 값과 퍼블릭 키를 바탕으로 v 값을 추정해서 완전한 서명을 만든 후 리턴
func (s *TxnSrv) getFullSignature(msg []byte, R []byte, S []byte, rightPubKey []byte) ([]byte, *errutil.ErrWrap) {
	vCandidates := [][]byte{{0}, {1}}

	sigWithRS := append(ethutil.PadLeftTo32Bytes(R), ethutil.PadLeftTo32Bytes(S)...)
	for _, v := range vCandidates {
		fullSig := append(sigWithRS, v...)
		recoverdPub, err := crypto.Ecrecover(msg, fullSig)
		if err != nil {
			return nil, errutil.NewErrWrap(500, "getFullSignature_crypto_ecrecover", err)
		}

		if hex.EncodeToString(recoverdPub) == hex.EncodeToString(rightPubKey) {
			return fullSig, nil
		}
	}

	return nil, errutil.NewErrWrap(500, "getFullSiganture", fmt.Errorf("failed to reconstruct public key from signature"))
}
