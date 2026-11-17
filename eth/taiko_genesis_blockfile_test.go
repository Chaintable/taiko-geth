package eth

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/Chaintable/pipeline/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestDebankGenesisMainnet(t *testing.T) {
	genesis := core.TaikoGenesisBlock(167000)
	block := genesis.ToBlock()
	file := buildGenesisBlockFile(block, genesis.Alloc)
	// Frozen mainnet alloc: 3 positive balances, 20 contracts, native token.
	if len(file.Txs) != 24 || len(file.Traces) != 24 || len(file.StorageContracts) != 16 {
		t.Fatalf("unexpected mainnet records: tx=%d trace=%d storage=%d", len(file.Txs), len(file.Traces), len(file.StorageContracts))
	}
	balances, codes := 0, 0
	seen := make(map[string]bool)
	for i, tx := range file.Txs {
		id, err := hex.DecodeString(strings.TrimPrefix(tx.ID, "0x"))
		if err != nil || len(id) != 32 || seen[tx.ID] {
			t.Fatalf("invalid/duplicate id %s", tx.ID)
		}
		seen[tx.ID] = true
		trace := file.Traces[i]
		if trace.TxID != tx.ID || trace.ID != util.ToHash([]string{tx.ID, "", "0"}) || tx.TransactionIndex != int64(i) {
			t.Fatalf("record linkage mismatch at %d", i)
		}
		if !tx.Status || tx.From != (common.Address{}).Hex() || tx.Gas.Sign() != 0 || tx.GasUsed.Sign() != 0 || tx.GasPrice.Sign() != 0 {
			t.Fatalf("invalid synthetic transaction at %d", i)
		}
		account := genesis.Alloc[common.HexToAddress(tx.To)]
		switch id[0] {
		case 1:
			balances++
			if (*big.Int)(tx.Value).Cmp(account.Balance) != 0 || trace.CallCreateType != "call" || trace.CallType != "call" || len(tx.Input) != 0 {
				t.Fatalf("balance record mismatch at %s", tx.To)
			}
		case 2:
			codes++
			if len(tx.Input) == 0 || !bytes.Equal(tx.Input, account.Code) || !bytes.Equal(trace.Output, account.Code) || trace.CallCreateType != "create" {
				t.Fatalf("code record mismatch at %s", tx.To)
			}
		case 3:
			if i != len(file.Txs)-1 || tx.To != "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" || len(tx.Input) != 0 || len(trace.Output) != 0 || trace.CallCreateType != "create" {
				t.Fatal("invalid native token record")
			}
		default:
			t.Fatalf("unexpected kind %d", id[0])
		}
		if (*big.Int)(trace.Value).Cmp((*big.Int)(tx.Value)) != 0 || !bytes.Equal(trace.Input, tx.Input) {
			t.Fatal("trace value/input mismatch")
		}
	}
	if balances != 3 || codes != 20 {
		t.Fatalf("balance/code counts %d/%d", balances, codes)
	}
	// Processing time is observational metadata, not deterministic chain data.
	file.Block.ProcessStartTimestamp = 0
	baseline, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		next := buildGenesisBlockFile(block, genesis.Alloc)
		next.Block.ProcessStartTimestamp = 0
		repeated, err := json.Marshal(next)
		if err != nil || !bytes.Equal(baseline, repeated) {
			t.Fatal("nondeterministic output")
		}
	}
}

func TestDebankGenesisEmptyAlloc(t *testing.T) {
	block := types.NewBlockWithHeader(&types.Header{Number: new(big.Int), Difficulty: new(big.Int), BaseFee: new(big.Int)})
	file := buildGenesisBlockFile(block, types.GenesisAlloc{
		common.HexToAddress("0x01"): {Balance: new(big.Int)},
		common.HexToAddress("0x02"): {},
	})
	if len(file.Txs) != 1 || len(file.Traces) != 1 || len(file.StorageContracts) != 0 {
		t.Fatal("zero/nil balances or empty code generated records")
	}
}
