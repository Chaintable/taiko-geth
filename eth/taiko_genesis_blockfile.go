package eth

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"strings"

	ptypes "github.com/Chaintable/pipeline/types"
	"github.com/Chaintable/pipeline/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// buildGenesisBlockFile represents alloc state as synthetic transactions and traces.
// Genesis initialization does not execute transactions, so replay hooks cannot
// produce these records. IDs use the bytes32 genesis kind/address convention.
func buildGenesisBlockFile(block *types.Block, alloc types.GenesisAlloc) *ptypes.BlockFile {
	file := &ptypes.BlockFile{
		Block:            util.BuildPipelineBlock(block),
		Txs:              make([]ptypes.Transaction, 0),
		Traces:           make([]ptypes.Trace, 0),
		Events:           make([]ptypes.Event, 0),
		ErrorEvents:      make([]ptypes.Event, 0),
		ErrorTraces:      make([]ptypes.Trace, 0),
		StorageContracts: make([]string, 0),
	}
	zeroAddr := "0x0000000000000000000000000000000000000000"
	appendRecord := func(kind int, addr string, value *big.Int, code []byte) {
		id := fmt.Sprintf("0x%02d%022d%s", kind, 0, strings.TrimPrefix(addr, "0x"))
		input := append(hexutil.Bytes{}, code...)
		file.Txs = append(file.Txs, ptypes.Transaction{
			ID:               id,
			From:             zeroAddr,
			To:               addr,
			Gas:              new(big.Int),
			GasPrice:         new(big.Int),
			GasUsed:          new(big.Int),
			Status:           true,
			GasFeeCap:        new(big.Int),
			GasTipCap:        new(big.Int),
			Input:            input,
			Nonce:            new(big.Int),
			TransactionIndex: int64(len(file.Txs)),
			Value:            (*hexutil.Big)(new(big.Int).Set(value)),
		})
		typ, callType := "create", ""
		if kind == 1 {
			typ, callType = "call", "call"
		}
		file.Traces = append(file.Traces, ptypes.Trace{
			ID:             util.ToHash([]string{id, "", "0"}),
			TxID:           id,
			From:           zeroAddr,
			To:             addr,
			Gas:            new(big.Int),
			GasUsed:        new(big.Int),
			Value:          (*hexutil.Big)(new(big.Int).Set(value)),
			Input:          input,
			Output:         append(hexutil.Bytes{}, code...),
			CallCreateType: typ,
			CallType:       callType,
			TraceAddress:   []int64{},
		})
	}
	addresses := make([]common.Address, 0, len(alloc))
	for addr := range alloc {
		addresses = append(addresses, addr)
	}
	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i][:], addresses[j][:]) < 0
	})
	for _, addr := range addresses {
		account := alloc[addr]
		address := strings.ToLower(addr.Hex())
		if len(account.Storage) > 0 {
			file.StorageContracts = append(file.StorageContracts, address)
		}
		if account.Balance != nil && account.Balance.Sign() > 0 {
			appendRecord(1, address, account.Balance, nil)
		}
		if len(account.Code) > 0 {
			appendRecord(2, address, new(big.Int), account.Code)
		}
	}
	appendRecord(3, "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", new(big.Int), nil)
	return file
}
