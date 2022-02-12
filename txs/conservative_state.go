package txs

import (
	"errors"
	"fmt"

	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/svm"
)

var (
	errBadNonce            = errors.New("bad nonce")
	errInsufficientBalance = errors.New("insufficient balance")
)

type ConservativeState struct {
	*svm.SVM

	logger log.Log
	pool   *TxPool
}

func NewConservativeState(state *svm.SVM, pool *TxPool, logger log.Log) *ConservativeState {
	return &ConservativeState{
		SVM:    state,
		pool:   pool,
		logger: logger,
	}
}

func (cs *ConservativeState) GetState(addr types.Address) (uint64, uint64) {
	return cs.GetNonce(addr), cs.GetBalance(addr)
}

// SelectTXsForProposal picks a specific number of random txs for miner.
func (cs *ConservativeState) SelectTXsForProposal(numOfTxs int) ([]types.TransactionID, []*types.Transaction, error) {
	txIds, err := cs.pool.getMempoolCandidateTXs(numOfTxs, cs.GetState)
	if err != nil {
		return nil, nil, err
	}

	if len(txIds) <= numOfTxs {
		return txIds, cs.pool.getTxByIds(txIds), nil
	}

	var ret []types.TransactionID
	for idx := range getRandIdxs(numOfTxs, len(txIds)) {
		// noinspection GoNilness
		ret = append(ret, txIds[idx])
	}

	return ret, cs.pool.getTxByIds(ret), nil
}

func (cs *ConservativeState) GetProjection(addr types.Address) (uint64, uint64, error) {
	nonce, balance := cs.GetState(addr)
	return cs.pool.GetProjection(addr, nonce, balance, MeshAndMempool)
}

func (cs *ConservativeState) validateNonceAndBalance(tx *types.Transaction) error {
	origin := tx.Origin()
	nonce, balance, err := cs.GetProjection(origin)
	if err != nil {
		return fmt.Errorf("failed to project state for account %v: %v", origin.Short(), err)
	}
	if tx.AccountNonce != nonce {
		return fmt.Errorf("%w: expected: %d, actual: %d", errBadNonce, nonce, tx.AccountNonce)
	}
	if (tx.Amount + tx.GetFee()) > balance { // TODO: Fee represents the absolute fee here, as a temporarily hack
		return fmt.Errorf("%w: available: %d, want to spend: %d[amount]+%d[fee]=%d",
			errInsufficientBalance, balance, tx.Amount, tx.GetFee(), tx.Amount+tx.GetFee())
	}
	return nil
}

// AddTxToMempool adds the provided transaction to the mempool after checking nonce and balance.
func (cs *ConservativeState) AddTxToMempool(tx *types.Transaction, checkValidity bool) error {
	if checkValidity {
		if err := cs.validateNonceAndBalance(tx); err != nil {
			return err
		}
	} else {
		if err := cs.pool.markTransactionDeleted(tx.ID()); err != nil {
			return err
		}
	}
	cs.pool.Put(tx.ID(), tx)
	return nil
}

func (cs *ConservativeState) GetTXFromMempool(tid types.TransactionID) (*types.MeshTransaction, error) {
	return cs.pool.Get(tid)
}

func (cs *ConservativeState) GetMeshTransaction(tid types.TransactionID) (*types.MeshTransaction, error) {
	tx, err := cs.pool.Get(tid)
	if err == nil {
		return tx, nil
	}
	return cs.pool.GetFromDB(tid)
}

// GetTransactions retrieves a list of txs by their id's.
func (cs *ConservativeState) GetTransactions(transactions []types.TransactionID) ([]*types.Transaction, map[types.TransactionID]struct{}) {
	missing := make(map[types.TransactionID]struct{})
	txs := make([]*types.Transaction, 0, len(transactions))
	for _, tid := range transactions {
		var (
			mtx *types.MeshTransaction
			err error
		)
		if mtx, err = cs.GetMeshTransaction(tid); err != nil {
			cs.logger.With().Warning("could not get tx", tid, log.Err(err))
			missing[tid] = struct{}{}
		} else {
			txs = append(txs, &mtx.Transaction)
		}
	}
	return txs, missing
}

func (cs *ConservativeState) ApplyLayer(lid types.LayerID, bid types.BlockID, txIDs []types.TransactionID, rewardByMiner map[types.Address]uint64) ([]*types.Transaction, error) {
	txs, missing := cs.pool.GetTransactionsFromDB(txIDs)
	if len(missing) > 0 {
		return nil, fmt.Errorf("find txs %v for applying layer %v", missing, lid)
	}
	// TODO: should miner IDs be sorted in a deterministic order prior to applying rewards?
	failedTxs, svmErr := cs.SVM.ApplyLayer(lid, txs, rewardByMiner)
	if svmErr != nil {
		cs.logger.With().Error("failed to apply txs",
			lid,
			log.Int("num_failed_txs", len(failedTxs)),
			log.Err(svmErr))
		// TODO: We want to panic here once we have a way to "remember" that we didn't apply these txs
		//  e.g. persist the last layer transactions were applied from and use that instead of `oldVerified`
		return failedTxs, fmt.Errorf("apply layer: %w", svmErr)
	}

	if err := cs.pool.updateTXWithBlockID(lid, bid, txs...); err != nil {
		cs.logger.With().Error("failed to update tx block ID in db", log.Err(err))
		return nil, err
	}
	for _, tx := range txs {
		if err := cs.pool.markTransactionApplied(tx.ID()); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Invalidate removes transaction from pool.
func (cs *ConservativeState) Invalidate(id types.TransactionID) {
	cs.pool.Invalidate(id)
}

// GetTransactionsByAddress retrieves txs for a single address in between layers [from, to].
// Guarantees that transaction will appear exactly once, even if origin and recipient is the same, and in insertion order.
func (cs *ConservativeState) GetTransactionsByAddress(from, to types.LayerID, address types.Address) ([]*types.MeshTransaction, error) {
	return cs.pool.GetTransactionsByAddress(from, to, address)
}
