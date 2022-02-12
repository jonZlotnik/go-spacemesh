package txs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/spacemeshos/go-spacemesh/sql/transactions"

	"github.com/spacemeshos/go-spacemesh/sql"

	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/events"
	"github.com/spacemeshos/go-spacemesh/rand"
)

type ProjectionMode uint

const (
	Mesh ProjectionMode = iota
	MeshAndMempool
)

// TxPool is a struct that holds txs received via gossip network.
type TxPool struct {
	db       *sql.Database
	mu       sync.RWMutex
	txs      map[types.TransactionID]*types.Transaction
	accounts map[types.Address]*AccountPendingTxs
}

// NewTxPool returns a new TxPool struct.
func NewTxPool(db *sql.Database) *TxPool {
	return &TxPool{
		db:       db,
		txs:      make(map[types.TransactionID]*types.Transaction),
		accounts: make(map[types.Address]*AccountPendingTxs),
	}
}

// Get returns transaction by provided id, it returns an error if transaction is not found.
func (tp *TxPool) Get(id types.TransactionID) (*types.MeshTransaction, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if tx, found := tp.txs[id]; found {
		return &types.MeshTransaction{
			Transaction: *tx,
		}, nil
	}

	return nil, errors.New("tx not in mempool")
}

// GetTransactionsByAddress retrieves txs for a single address in beetween layers [from, to].
// Guarantees that transaction will appear exactly once, even if origin and recipient is the same, and in insertion order.
func (tp *TxPool) GetTransactionsByAddress(from, to types.LayerID, address types.Address) ([]*types.MeshTransaction, error) {
	return transactions.FilterByAddress(tp.db, from, to, address)
}

func (tp *TxPool) markTransactionApplied(tid types.TransactionID) error {
	if err := transactions.Applied(tp.db, tid); err != nil {
		return err
	}
	return nil
}

func (tp *TxPool) markTransactionDeleted(tid types.TransactionID) error {
	if err := transactions.MarkDeleted(tp.db, tid); err != nil && !errors.Is(err, sql.ErrNotFound) {
		return err
	}
	return nil
}

func (tp *TxPool) updateTXWithBlockID(lid types.LayerID, bid types.BlockID, txs ...*types.Transaction) error {
	return tp.writeTransactions(lid, bid, txs...)
}

// writeTransactions writes all transactions associated with a block atomically.
func (tp *TxPool) writeTransactions(layerID types.LayerID, bid types.BlockID, txs ...*types.Transaction) error {
	dbtx, err := tp.db.Tx(context.Background())
	if err != nil {
		return err
	}
	defer dbtx.Release()
	for _, tx := range txs {
		if err := transactions.Add(dbtx, layerID, bid, tx); err != nil {
			return err
		}
	}
	return dbtx.Commit()
}

func (tp *TxPool) GetTransactionsFromDB(transactions []types.TransactionID) ([]*types.Transaction, map[types.TransactionID]struct{}) {
	missing := make(map[types.TransactionID]struct{})
	txs := make([]*types.Transaction, 0, len(transactions))
	for _, id := range transactions {
		if tx, err := tp.GetFromDB(id); err != nil {
			missing[id] = struct{}{}
		} else {
			txs = append(txs, &tx.Transaction)
		}
	}
	return txs, missing
}

func (tp *TxPool) GetFromDB(id types.TransactionID) (*types.MeshTransaction, error) {
	mtx, err := transactions.Get(tp.db, id)
	if err != nil {
		return nil, errors.New("tx not in db")
	}
	return mtx, nil
}

func (tp *TxPool) getTxByIds(txsIDs []types.TransactionID) []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(txsIDs))
	for _, tx := range txsIDs {
		txs = append(txs, tp.txs[tx])
	}
	return txs
}

func getRandIdxs(numOfTxs, spaceSize int) map[uint64]struct{} {
	idxs := make(map[uint64]struct{})
	for len(idxs) < numOfTxs {
		rndInt := rand.Uint64()
		idx := rndInt % uint64(spaceSize)
		idxs[idx] = struct{}{}
	}
	return idxs
}

// Put inserts a transaction into the mem pool. It indexes it by source and dest addresses as well.
func (tp *TxPool) Put(id types.TransactionID, tx *types.Transaction) {
	tp.put(id, tx)
	events.ReportNewTx(types.LayerID{}, tx)
	events.ReportAccountUpdate(tx.Origin())
	events.ReportAccountUpdate(tx.GetRecipient())
}

func (tp *TxPool) put(id types.TransactionID, tx *types.Transaction) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.txs[id] = tx
	tp.getOrCreate(tx.Origin()).Add(types.LayerID{}, tx)
}

// Invalidate removes transaction from pool.
func (tp *TxPool) Invalidate(id types.TransactionID) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if tx, found := tp.txs[id]; found {
		if pendingTxs, found := tp.accounts[tx.Origin()]; found {
			// Once a tx appears in a block we want to invalidate all of this nonce's variants. The mempool currently
			// only accepts one version, but this future-proofs it.
			pendingTxs.RemoveNonce(tx.AccountNonce, func(id types.TransactionID) {
				delete(tp.txs, id)
			})
			if pendingTxs.IsEmpty() {
				delete(tp.accounts, tx.Origin())
			}
			// We only report those transactions that are being dropped from the txPool here as
			// conflicting since they won'tp be reported anywhere else. There is no need to report
			// the initial tx here since it'll be reported as part of a new block/layer anyway.
			events.ReportTxWithValidity(types.LayerID{}, tx, false)
		}
	}
}

func (tp *TxPool) getDBProjection(addr types.Address, prevNonce, prevBalance uint64) (uint64, uint64, error) {
	txs, err := transactions.FilterPending(tp.db, addr)
	if err != nil {
		return 0, 0, fmt.Errorf("db get pending txs: %w", err)
	}

	pending := NewAccountPendingTxs()
	for _, tx := range txs {
		pending.Add(tx.LayerID, &tx.Transaction)
	}
	nonce, balance := pending.GetProjection(prevNonce, prevBalance)
	return nonce, balance, nil
}

func (tp *TxPool) getMemPoolProjection(addr types.Address, prevNonce, prevBalance uint64) (uint64, uint64) {
	tp.mu.RLock()
	account, found := tp.accounts[addr]
	tp.mu.RUnlock()
	if !found {
		return prevNonce, prevBalance
	}

	return account.GetProjection(prevNonce, prevBalance)
}

// GetProjection returns the estimated nonce and balance for the provided address addr and previous nonce and balance
// projecting state is done by applying transactions from the pool.
func (tp *TxPool) GetProjection(addr types.Address, prevNonce, prevBalance uint64, mode ProjectionMode) (uint64, uint64, error) {
	nonce, balance, err := tp.getDBProjection(addr, prevNonce, prevBalance)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get db projection: %w", err)
	}
	if mode == Mesh {
		return nonce, balance, err
	}
	nonce, balance = tp.getMemPoolProjection(addr, nonce, balance)
	return nonce, balance, nil
}

// ⚠️ must be called under write-lock.
func (tp *TxPool) getOrCreate(addr types.Address) *AccountPendingTxs {
	account, found := tp.accounts[addr]
	if !found {
		account = NewAccountPendingTxs()
		tp.accounts[addr] = account
	}
	return account
}

func (tp *TxPool) getMempoolCandidateTXs(numTXs int, getState func(types.Address) (uint64, uint64)) ([]types.TransactionID, error) {
	txIds := make([]types.TransactionID, 0, numTXs)

	tp.mu.RLock()
	defer tp.mu.RUnlock()
	for addr, account := range tp.accounts {
		nonce, balance := getState(addr)
		nonce, balance, err := tp.GetProjection(addr, nonce, balance, Mesh)
		if err != nil {
			return nil, fmt.Errorf("get projection for addr %s: %v", addr.Short(), err)
		}
		accountTxIds, _, _ := account.ValidTxs(nonce, balance)
		txIds = append(txIds, accountTxIds...)
	}
	return txIds, nil
}
