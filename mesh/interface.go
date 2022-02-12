package mesh

import (
	"context"

	"github.com/spacemeshos/go-spacemesh/common/types"
)

//go:generate mockgen -package=mocks -destination=./mocks/mocks.go -source=./interface.go

type conservativeState interface {
	ApplyLayer(types.LayerID, types.BlockID, []types.TransactionID, map[types.Address]uint64) ([]*types.Transaction, error)
	GetStateRoot() types.Hash32
	Rewind(types.LayerID) (types.Hash32, error)
	AddTxToMempool(*types.Transaction, bool) error
	Invalidate(types.TransactionID)
	GetMeshTransaction(types.TransactionID) (*types.MeshTransaction, error)
	//GetTransactions([]types.TransactionID) ([]*types.Transaction, map[types.TransactionID]struct{})
	//
	//AddressExists(types.Address) bool
	//GetAllAccounts() (*types.MultipleAccountsState, error)
	//GetBalance(types.Address) uint64
	//GetLayerApplied(types.TransactionID) *types.LayerID
	//GetLayerStateRoot(types.LayerID) (types.Hash32, error)
	//GetNonce(types.Address) uint64
}

type tortoise interface {
	OnBallot(*types.Ballot)
	OnBlock(*types.Block)
	HandleIncomingLayer(context.Context, types.LayerID) (oldPbase, newPbase types.LayerID, reverted bool)
}
