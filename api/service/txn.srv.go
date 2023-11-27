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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

type TxnSrv struct {
	chainID *big.Int
	kmsSrv  *KmsSrv
}

type AccessListTxnWithoutSig struct {
	ChainID    *big.Int `rlp:"nil"`
	Nonce      uint64
	GasPrice   *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList
	R          *big.Int `rlp:"optional"`
	S          *big.Int `rlp:"optional"`
	V          *big.Int `rlp:"optional"`
}

type DynamicFeeTxnWithoutSig struct {
	ChainID    *big.Int `rlp:"nil"`
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList
	R          *big.Int `rlp:"optional"`
	S          *big.Int `rlp:"optional"`
	V          *big.Int `rlp:"optional"`
}

type BlobTxnWithoutSig struct {
	ChainID    *uint256.Int `rlp:"nil"`
	Nonce      uint64
	GasTipCap  *uint256.Int
	GasFeeCap  *uint256.Int
	Gas        uint64
	To         common.Address `rlp:"nil"`
	Value      *uint256.Int
	Data       []byte
	AccessList types.AccessList
	BlobFeeCap *uint256.Int
	BlobHashes []common.Hash
	R          *big.Int `rlp:"optional"`
	S          *big.Int `rlp:"optional"`
	V          *big.Int `rlp:"optional"`
}

func NewTxnSrv(chainID *big.Int, kmsSrv *KmsSrv) *TxnSrv {
	return &TxnSrv{chainID, kmsSrv}
}

// 서명되지 않은 트렌젝션을 받아서, 서명한뒤 리턴
func (s *TxnSrv) SignSerializedTxn(txnDTO *dto.TxnDTO) (*res.SingedTxnRes, *errutil.ErrWrap) {
	parsedTxn, err := s.parseTxn(txnDTO.SerializedTxn)
	if err != nil {
		return nil, errutil.NewErrWrap(400, "", err)
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
	// 최종V = {0,1} + CHAIN_ID * 2 + 35

	byteSignedTxn, err := signedTxn.MarshalBinary()
	if err != nil {
		return nil, errutil.NewErrWrap(500, "SignSerializedTxn_types.transaction_marshalBinary", err)
	}
	return &res.SingedTxnRes{SignedTxn: "0x" + common.Bytes2Hex(byteSignedTxn)}, nil
}

// 직렬화된 트렌젝션 데이터를 type.Transaction Struct로 변환
func (s *TxnSrv) parseTxn(serializedTxn string) (*types.Transaction, error) {
	txBytes := common.FromHex(serializedTxn)

	// legacy 를 제외한 typed Transaction은 서명값(r, s, v)이 없는상태로 파싱해버리면 rlp: too few elements 에러를 뱉는다
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
		var inner AccessListTxnWithoutSig
		err := rlp.DecodeBytes(txBytes[1:], &inner)
		if err != nil {
			return nil, err
		}
		fmt.Println("inner chainID", inner.ChainID)
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
		var inner DynamicFeeTxnWithoutSig
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
		var inner BlobTxnWithoutSig
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

// 02f86b800101841be5aec382892794a90ba7566dd8f2acda2c3af11bf426c569b7f74780b844a9059cbb000000000000000000000000acfe053e5d0c0fa206c53f09c3cc801301df3cef000000000000000000000000000000000000000000000001158e460913d00000c0808080
// 02f87586059407ad8e8b818885ba43b7400085ba43b74000829217948b1c97e058e921d41f7abffc1099f071a2bb30c380b844a9059cbb0000000000000000000000005f90d10443b03f46a6c3513fe62f60733e7bcea70000000000000000000000000000000000000000000000008ac7230489e80000c0808080
// 02f87886059407ad8e8b818885ba43b7400085ba43b74000829217948b1c97e058e921d41f7abffc1099f071a2bb30c380b844a9059cbb0000000000000000000000005f90d10443b03f46a6c3513fe62f60733e7bcea70000000000000000000000000000000000000000000000008ac7230489e80000c0808080
