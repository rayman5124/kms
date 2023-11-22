package srv

import (
	"context"
	"encoding/hex"
	"fmt"
	"kms/tutorial/api/model/dto"
	"kms/tutorial/api/model/res"
	"kms/tutorial/common/utils/errutil"
	"kms/tutorial/common/utils/ethutil"
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TxnSrv struct {
	client  *ethclient.Client
	ChainID *big.Int
	kmsSrv  *KmsSrv
}

func NewTxnSrv(rpcEndPoint string, kmSrv *KmsSrv) *TxnSrv {
	client := errutil.HandleFatal(ethclient.Dial(rpcEndPoint))
	chainID := errutil.HandleFatal(client.ChainID(context.TODO()))
	return &TxnSrv{client, chainID, kmSrv}
}

// 직렬화된 트렌젝션을 전송
func (s *TxnSrv) SendSerializedTxn(txnDTO *dto.TxnDTO) (*res.TxnRes, *errutil.ErrWrap) {
	parsedTxn, errWrap := s.parseTxn(txnDTO.SerializedTxn)
	if errWrap != nil {
		return nil, errWrap
	}

	hash, errWrap := s.SignAndSendTxn(txnDTO.KeyID, parsedTxn)
	if errWrap != nil {
		return nil, errWrap
	}

	return &res.TxnRes{Hash: hash.String()}, nil
}

// 서명되지 않은 트렌젝션을 받아서, 서명 및 제출 후 제출된 트래젝션 해시를 반환
func (s *TxnSrv) SignAndSendTxn(keyID string, txn *types.Transaction) (*common.Hash, *errutil.ErrWrap) {
	sender, errWrap := s.kmsSrv.GetAccount(&dto.AccountDTO{KeyID: keyID})
	if errWrap != nil {
		return nil, errWrap
	}

	// verify transaction
	err := s.verifyTxn(txn, common.HexToAddress(sender.Address))
	if err != nil {
		return nil, errutil.NewErrWrap(400, "", err)
	}

	signer := types.NewCancunSigner(s.ChainID)
	txnMsg := signer.Hash(txn).Bytes()

	// kms로부터 서명을 받아온다
	R, S, errWrap := s.kmsSrv.Sign(keyID, txnMsg)
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

	pubKey, errWrap := s.kmsSrv.GetPubkey(&dto.AccountDTO{KeyID: keyID})
	if errWrap != nil {
		return nil, errWrap
	}

	// V 값을 유추해서 완전한 이더리움 서명을 만든다
	signature, errWrap := s.getFullSignature(txnMsg, R, S, pubKey)
	if errWrap != nil {
		return nil, errWrap
	}

	signedTxn, err := txn.WithSignature(signer, signature)
	if err != nil {
		return nil, errutil.NewErrWrap(500, "SignAndSendTxn_types.transction_withSignature", err)
	}

	if err := s.client.SendTransaction(context.TODO(), signedTxn); err != nil {
		return nil, errutil.NewErrWrap(400, "", err)
	}

	hash := signedTxn.Hash()
	return &hash, nil
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

	return nil, errutil.NewErrWrap(500, "getFullSiganture", errors.New("failed to reconstruct public key from signature"))
}

func (s *TxnSrv) verifyTxn(txn *types.Transaction, sender common.Address) error {
	callmsg := map[string]interface{}{
		"from":  sender,
		"to":    txn.To(),
		"value": txn.Value(),
		"data":  hexutil.Bytes(txn.Data()),
	}

	var result interface{}
	if err := s.client.Client().Call(&result, "eth_call", callmsg, "latest"); err != nil {
		fmt.Printf("call error: %+v\n ", err)
		return err
	}

	return nil
}

// 직랼화된 트렌젝션 데이터를 type.Transaction Struct로 변환
func (s *TxnSrv) parseTxn(serializedTxn string) (*types.Transaction, *errutil.ErrWrap) {
	var parsedTxn types.Transaction

	cutPrefix, _ := strings.CutPrefix(serializedTxn, "0x")

	err := parsedTxn.UnmarshalBinary(common.Hex2Bytes(cutPrefix))
	if err != nil {
		return nil, errutil.NewErrWrap(400, "", err)
	}

	return &parsedTxn, nil
}

func (s *TxnSrv) AutoFillTxn(txn *types.LegacyTx, sender common.Address) (*types.Transaction, *errutil.ErrWrap) {

	if txn.Nonce == 0 {
		nonce, err := s.client.PendingNonceAt(context.Background(), sender)
		if err != nil {
			return nil, errutil.NewErrWrap(500, "AutoFillTxn_ethClient_pendingNonceAt", err)
		}
		txn.Nonce = nonce
	}

	if txn.GasPrice == nil {
		gasPrice, err := s.client.SuggestGasPrice(context.TODO())
		if err != nil {
			return nil, errutil.NewErrWrap(500, "AutoFillTxn_ethClient_suggestGasPrice", err)
		}
		txn.GasPrice = gasPrice
	}

	if txn.Gas == 0 {
		gas, err := s.client.EstimateGas(context.TODO(), ethereum.CallMsg{
			To:    txn.To,
			From:  sender,
			Data:  txn.Data,
			Value: txn.Value,
		})
		if err != nil {
			return nil, errutil.NewErrWrap(400, "", err)
		}
		txn.Gas = gas
	}

	return types.NewTx(txn), nil
}
