package srv

import (
	"bytes"
	"fmt"
	"kms/wallet/app/api/model/dto"
	"kms/wallet/app/api/model/res"
	"kms/wallet/common/errwrap"
	"kms/wallet/common/utils/ethutil"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

type TxnSrv struct {
	chainID *big.Int
	kmsSrv  *KmsSrv
}

type AccessListTxnOptionalSig struct {
	ChainID    *big.Int
	Nonce      uint64
	GasPrice   *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList
	V, R, S    *big.Int `rlp:"optional"`
}

type DynamicFeeTxnOptionalSig struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList
	V, R, S    *big.Int `rlp:"optional"`
}

type BlobTxnOptionalSig struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int
	GasFeeCap  *uint256.Int
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList types.AccessList
	BlobFeeCap *uint256.Int
	BlobHashes []common.Hash
	Sidecar    *types.BlobTxSidecar `rlp:"-"`
	V, R, S    *big.Int             `rlp:"optional"`
}

func NewTxnSrv(chainID *big.Int, kmsSrv *KmsSrv) *TxnSrv {
	return &TxnSrv{chainID, kmsSrv}
}

// 서명되지 않은 트렌젝션을 받아서, 서명한뒤 리턴
func (s *TxnSrv) SignSerializedTxn(txnDTO *dto.SerializedTxnDTO) (*res.SingedTxnRes, *errwrap.ErrWrap) {
	// 퍼블릭 키에 대한 요청 먼저 고루틴으로
	var (
		pubKey        []byte
		pubKeyErrChan = make(chan *errwrap.ErrWrap)
	)
	go func() {
		pubKeyRet, errWrap := s.kmsSrv.GetPubkey(&dto.KeyIdDTO{KeyID: txnDTO.KeyID})
		if errWrap != nil {
			pubKeyErrChan <- errWrap.ChangeCode(500).AddLayer("SignSerializedTxn", "KmsSrv")
		} else {
			pubKey = pubKeyRet
			pubKeyErrChan <- nil
		}
	}()

	parsedTxn, err := s.parseTxn(txnDTO.SerializedTxn)
	if err != nil {
		return nil, errwrap.ClientErr(err)
	}

	signer := types.NewCancunSigner(s.chainID)
	txnMsg := signer.Hash(parsedTxn).Bytes()

	// kms로부터 서명을 받아온다
	R, S, errWrap := s.kmsSrv.Sign(txnDTO.KeyID, txnMsg)
	if errWrap != nil {
		return nil, errWrap.AddLayer("SignSerializedTxn", "KmsSrv")
	}

	// S값 가공
	secp256k1n := crypto.S256().Params().N                        // secp256k1 타원곡선의 최댓값
	halfSecp256k1n := new(big.Int).Div(secp256k1n, big.NewInt(2)) // 타원곡선의 최대값의 절반
	// S 값이 타원곡선 최댓값의 절반보다 크면 변환해서 사용 (reference -> EIP2 https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md)
	if sBigInt := new(big.Int).SetBytes(S); sBigInt.Cmp(halfSecp256k1n) > 0 {
		// 원래 ECDSA 서명 방식에서 기존 S, curve.n - S 둘다 유효한 값이지만 이더리움에서는 후자만 유효하다
		S = new(big.Int).Sub(secp256k1n, sBigInt).Bytes()
	}

	// V 값을 유추해서 완전한 이더리움 서명을 만든다
	if pubKeyErr := <-pubKeyErrChan; pubKeyErr != nil {
		return nil, pubKeyErr
	}
	signature, errWrap := s.getFullSignature(txnMsg, R, S, pubKey)
	if errWrap != nil {
		return nil, errWrap.AddLayer("SignSerializedTxn", "TxnSrv")
	}

	signedTxn, err := parsedTxn.WithSignature(signer, signature)
	if err != nil {
		return nil, errwrap.ServerErr(err).AddLayer("SignSerializedTxn", "tyes.Transaction", "WithSignature")
	}
	// 최종 V = {0,1} + CHAIN_ID * 2 + 35

	byteSignedTxn, err := signedTxn.MarshalBinary()
	if err != nil {
		return nil, errwrap.ServerErr(err).AddLayer("SignSerializedTxn", "types.Transaction", "MarshalBinary")
	}
	return &res.SingedTxnRes{SignedTxn: "0x" + common.Bytes2Hex(byteSignedTxn)}, nil
}

// 직렬화된 트렌젝션 데이터를 type.Transaction Struct로 변환
func (s *TxnSrv) parseTxn(serializedTxn string) (*types.Transaction, error) {
	txBytes := common.FromHex(serializedTxn)

	// legacy 를 제외한 typed Transaction은 서명값(r, s, v)이 필수이기 때문에 서명이 없는 serialized txn을 파싱할때 에러가 발생한다.
	// go-ethereum 의 types 패키지를 활용하여 typed Transaction을 생성하면 자동으로 default 서명이 들어가지만 (r:0, s:0, v:0)
	// Type-script의 ethers를 통해 typed Transaction을 생성하면 사인전에 서명값이 아예 존재하지 않아서 바로 파싱해버리면 에러가 발생한다.
	// 따라서 트렌젝션 타입별로 따로 처리를 해준다

	// legacy Txn
	if len(txBytes) > 0 && txBytes[0] > 0x7f {
		var parsedTxn types.Transaction
		err := parsedTxn.UnmarshalBinary(txBytes)
		if err != nil {
			return nil, err
		}
		return &parsedTxn, nil
	}

	// typed Txn
	if len(txBytes) <= 1 {
		return nil, fmt.Errorf("typed transaction too short")
	}
	switch txBytes[0] { // 0번째 인덱스에는 트렌젝션 타입에 대한 정보가 담겨있다.
	case types.AccessListTxType:
		var inner AccessListTxnOptionalSig
		err := rlp.DecodeBytes(txBytes[1:], &inner)
		if err != nil {
			return nil, err
		}
		return types.NewTx(&types.AccessListTx{
			ChainID:    s.chainID,
			Nonce:      inner.Nonce,
			GasPrice:   inner.GasPrice,
			Gas:        inner.Gas,
			To:         inner.To,
			Value:      inner.Value,
			Data:       inner.Data,
			AccessList: inner.AccessList,
		}), nil

	case types.DynamicFeeTxType:
		var inner DynamicFeeTxnOptionalSig
		err := rlp.DecodeBytes(txBytes[1:], &inner)
		if err != nil {
			return nil, err
		}
		return types.NewTx(&types.DynamicFeeTx{
			ChainID:    s.chainID,
			Nonce:      inner.Nonce,
			GasTipCap:  inner.GasTipCap,
			GasFeeCap:  inner.GasFeeCap,
			Gas:        inner.Gas,
			To:         inner.To,
			Value:      inner.Value,
			Data:       inner.Data,
			AccessList: inner.AccessList,
		}), nil

	case types.BlobTxType:
		var inner BlobTxnOptionalSig
		err := rlp.DecodeBytes(txBytes[1:], &inner)
		if err != nil {
			return nil, err
		}
		chainID, _ := uint256.FromBig(s.chainID)
		return types.NewTx(&types.BlobTx{
			ChainID:    chainID,
			Nonce:      inner.Nonce,
			GasTipCap:  inner.GasTipCap,
			GasFeeCap:  inner.GasFeeCap,
			Gas:        inner.Gas,
			To:         inner.To,
			Value:      inner.Value,
			Data:       inner.Data,
			AccessList: inner.AccessList,
			BlobFeeCap: inner.BlobFeeCap,
			BlobHashes: inner.BlobHashes,
		}), nil

	default:
		return nil, fmt.Errorf("unsuppported transaction type")
	}

}

// r, s 값과 퍼블릭 키를 바탕으로 v 값을 추정해서 완전한 서명을 만든 후 리턴
func (s *TxnSrv) getFullSignature(msg []byte, R []byte, S []byte, rightPubKey []byte) ([]byte, *errwrap.ErrWrap) {
	vCandidates := [][]byte{{0}, {1}}

	sigWithRS := append(ethutil.PadLeftTo32Bytes(R), ethutil.PadLeftTo32Bytes(S)...)
	for _, v := range vCandidates {
		fullSig := append(sigWithRS, v...)
		recoverdPub, err := crypto.Ecrecover(msg, fullSig)
		if err != nil {
			return nil, errwrap.ServerErr(err).AddLayer("getFullSignature", "crypto", "Encrecover")
		}

		if bytes.Equal(recoverdPub, rightPubKey) {
			return fullSig, nil
		}
	}

	return nil, errwrap.ServerErr(fmt.Errorf("failed to reconstruct public key from signature")).AddLayer("getFullSignature")
}
