package layerfetcher

import "github.com/spacemeshos/go-spacemesh/common/types"

// layerBlocks is the response for a given layer ID
type layerBlocks struct {
	// Blocks are the blocks in a layer
	Blocks []types.BlockID `ssz-max:"4096"`
	// InputVector is the input vector for verifying tortoise
	InputVector []types.BlockID `ssz-max:"4096"`
	// ProcessedLayer is the latest processed layer from peer
	ProcessedLayer types.LayerID
	// Hash is the hash of contextually valid blocks (sorted by block ID) in the given layer
	Hash types.Hash32
	// AggregatedHash is the aggregated hash of all layers up to the given layer
	AggregatedHash types.Hash32
}

type atxContainer struct {
	List []types.ATXID `ssz-max:"4096"`
}
