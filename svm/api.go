package svm

import (
	"fmt"

	"github.com/spacemeshos/go-spacemesh/api/config"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/common/util"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/svm/state"
)

// SVM is an entry point for all SVM operations.
type SVM struct {
	state *state.TransactionProcessor
	log   log.Logger
}

// New creates a new `SVM` instance from the given `state` and `logger`.
func New(allStates, processorDb database.Database, logger log.Log) *SVM {
	return &SVM{
		state: state.NewTransactionProcessor(allStates, processorDb, logger),
		log:   logger,
	}
}

// SetupGenesis creates new accounts and adds balances as dictated by `conf`.
func (svm *SVM) SetupGenesis(conf *config.GenesisConfig) error {
	if conf == nil {
		conf = config.DefaultGenesisConfig()
	}
	for id, balance := range conf.Accounts {
		bytes := util.FromHex(id)
		if len(bytes) == 0 {
			return fmt.Errorf("cannot decode entry %s for genesis account", id)
		}
		// just make it explicit that we want address and not a public key
		if len(bytes) != types.AddressLength {
			return fmt.Errorf("%s must be an address of size %d", id, types.AddressLength)
		}
		addr := types.BytesToAddress(bytes)
		svm.state.CreateAccount(addr)
		svm.state.AddBalance(addr, balance)
		svm.log.With().Info("genesis account created",
			log.String("address", addr.Hex()),
			log.Uint64("balance", balance))
	}

	_, err := svm.state.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit genesis state: %w", err)
	}
	return nil
}

// ApplyLayer applies the given rewards to some miners as well as a vector of
// transactions for the given layer. to miners vector for layer. It returns an
// error on failure, as well as a vector of failed transactions.
func (svm *SVM) ApplyLayer(layerID types.LayerID, transactions []*types.Transaction, rewards map[types.Address]uint64) ([]*types.Transaction, error) {
	svm.state.ApplyRewards(layerID, rewards)
	failedTxs, err := svm.state.ApplyTransactions(layerID, transactions)
	if err != nil {
		return failedTxs, fmt.Errorf("SVM couldn't apply layer %d: %w", layerID.Uint32(), err)
	}

	return failedTxs, nil
}

// AddressExists checks if an account address exists in this node's global state.
func (svm *SVM) AddressExists(addr types.Address) bool {
	return svm.state.AddressExists(addr)
}

// GetLayerApplied gets the layer id at which this tx was applied.
func (svm *SVM) GetLayerApplied(txID types.TransactionID) *types.LayerID {
	return svm.state.GetLayerApplied(txID)
}

// GetLayerStateRoot returns the state root at a given layer.
func (svm *SVM) GetLayerStateRoot(layer types.LayerID) (types.Hash32, error) {
	hash, err := svm.state.GetLayerStateRoot(layer)
	if err != nil {
		err = fmt.Errorf("SVM couldn't get the root hash of layer %d: %w", layer.Uint32(), err)
	}
	return hash, err
}

// GetStateRoot gets the current state root hash.
func (svm *SVM) GetStateRoot() types.Hash32 {
	return svm.state.GetStateRoot()
}

// Rewind loads the given layer state from persistent storage. On success, it
// also returns the current state root hash *after* rewinding.
func (svm *SVM) Rewind(layer types.LayerID) (types.Hash32, error) {
	err := svm.state.LoadState(layer)
	if err != nil {
		return types.Hash32{}, fmt.Errorf("SVM couldn't rewind back to layer %d: %w", layer.Uint32(), err)
	}
	return svm.state.GetStateRoot(), err
}

// GetBalance Retrieve the balance from the given address or 0 if object not found.
func (svm *SVM) GetBalance(addr types.Address) uint64 {
	return svm.state.GetBalance(addr)
}

// GetNonce gets the current nonce of the given addr, if the address is not
// found it returns 0.
func (svm *SVM) GetNonce(addr types.Address) uint64 {
	return svm.state.GetNonce(addr)
}

// GetAllAccounts returns a dump of all accounts in global state.
func (svm *SVM) GetAllAccounts() (*types.MultipleAccountsState, error) {
	accounts, err := svm.state.GetAllAccounts()
	if err != nil {
		err = fmt.Errorf("SVM couldn't get all accounts: %w", err)
	}
	return accounts, err
}
