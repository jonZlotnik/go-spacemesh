package tortoise

import (
	"context"
	"errors"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/mesh"
	"github.com/syndtr/goleveldb/leveldb"
	"time"
)

type blockDataProvider interface {
	ContextuallyValidBlock(types.LayerID) (map[types.BlockID]struct{}, error)
	GetBlock(types.BlockID) (*types.Block, error)
	LayerBlockIds(types.LayerID) ([]types.BlockID, error)
	LayerBlocks(types.LayerID) ([]*types.Block, error)

	GetCoinflip(context.Context, types.LayerID) (bool, bool)
	GetLayerInputVectorByID(types.LayerID) ([]types.BlockID, error)
	SaveContextualValidity(types.BlockID, types.LayerID, bool) error
	ContextualValidity(types.BlockID) (bool, error)

	Persist(key []byte, v interface{}) error
	Retrieve(key []byte, v interface{}) (interface{}, error)
}

var (
	errNoBaseBlockFound                 = errors.New("no good base block within exception vector limit")
	errBaseBlockNotInDatabase           = errors.New("inconsistent state: can't find base block in database")
	errNotSorted                        = errors.New("input blocks are not sorted by layerID")
	errstrNoCoinflip                    = "no weak coin value for current layer"
	errstrTooManyExceptions             = "too many exceptions to base block vote"
	errstrBaseBlockLayerMissing         = "base block layer not found"
	errstrBaseBlockNotFoundInLayer      = "base block opinions not found in layer"
	errstrConflictingVotes              = "conflicting votes found in block"
	errstrCantFindLayer                 = "inconsistent state: can't find layer in database"
	errstrUnableToCalculateLocalOpinion = "unable to calculate local opinion for layer"
)

func blockMapToArray(m map[types.BlockID]struct{}) []types.BlockID {
	arr := make([]types.BlockID, len(m))
	i := 0
	for b := range m {
		arr[i] = b
		i++
	}
	return types.SortBlockIDs(arr)
}

type turtle struct {
	logger log.Log

	bdp blockDataProvider

	// last layer processed: note that tortoise does not have a concept of "current" layer (and it's not aware of the
	// current time or latest tick). As far as Tortoise is concerned, Last is the current layer. This is a subjective
	// view of time, but Tortoise receives layers as soon as Hare finishes processing them or when they are received via
	// gossip, and there's nothing for Tortoise to verify without new data anyway.
	Last types.LayerID

	// last evicted layer
	LastEvicted types.LayerID

	// hare lookback (distance): up to Hdist layers back, we only consider hare results/input vector
	Hdist types.LayerID

	// hare abort distance: we wait up to Zdist layers for hare results/input vector, before invalidating a layer
	// without hare results.
	Zdist types.LayerID

	// the number of layers we wait until we have confidence that w.h.p. all honest nodes have reached consensus on the
	// contents of a layer
	ConfidenceParam types.LayerID

	// the size of the tortoise sliding window which controls how far back the tortoise stores data
	WindowSize types.LayerID

	// thresholds used for determining finality, and whether to use local or global results, respectively
	GlobalThreshold uint8
	LocalThreshold  uint8

	AvgLayerSize  int
	MaxExceptions int

	GoodBlocksIndex map[types.BlockID]struct{}

	Verified types.LayerID

	// this matrix stores the opinion of each block about other blocks. blocks are indexed by layer.
	// it stores good and bad blocks
	BlockOpinionsByLayer map[types.LayerID]map[types.BlockID]Opinion

	// how often we want to rerun from genesis
	RerunInterval time.Duration
}

// SetLogger sets the Log instance for this turtle
func (t *turtle) SetLogger(logger log.Log) {
	t.logger = logger
}

// newTurtle creates a new verifying tortoise algorithm instance
func newTurtle(
	bdp blockDataProvider,
	hdist,
	zdist,
	confidenceParam,
	windowSize,
	avgLayerSize int,
	globalThreshold,
	localThreshold uint8,
	rerun time.Duration,
) *turtle {
	return &turtle{
		logger:               log.NewDefault("trtl"),
		Hdist:                types.LayerID(hdist),
		Zdist:                types.LayerID(zdist),
		ConfidenceParam:      types.LayerID(confidenceParam),
		WindowSize:           types.LayerID(windowSize),
		GlobalThreshold:      globalThreshold,
		LocalThreshold:       localThreshold,
		bdp:                  bdp,
		Last:                 0,
		AvgLayerSize:         avgLayerSize,
		GoodBlocksIndex:      make(map[types.BlockID]struct{}),
		BlockOpinionsByLayer: make(map[types.LayerID]map[types.BlockID]Opinion, hdist),
		MaxExceptions:        hdist * avgLayerSize * 100,
		RerunInterval:        rerun,
	}
}

// cloneTurtle creates a new verifying tortoise instance using the params of this instance
func (t *turtle) cloneTurtle() *turtle {
	return newTurtle(
		t.bdp,
		int(t.Hdist),
		int(t.Zdist),
		int(t.ConfidenceParam),
		int(t.WindowSize),
		t.AvgLayerSize,
		t.GlobalThreshold,
		t.LocalThreshold,
		t.RerunInterval,
	)
}

func (t *turtle) init(ctx context.Context, genesisLayer *types.Layer) {
	// Mark the genesis layer as “good”
	t.logger.WithContext(ctx).With().Debug("initializing genesis layer for verifying tortoise",
		genesisLayer.Index(),
		genesisLayer.Hash().Field())
	t.BlockOpinionsByLayer[genesisLayer.Index()] = make(map[types.BlockID]Opinion)
	for _, blk := range genesisLayer.Blocks() {
		id := blk.ID()
		t.BlockOpinionsByLayer[genesisLayer.Index()][blk.ID()] = Opinion{
			BlockOpinions: make(map[types.BlockID]vec),
		}
		t.GoodBlocksIndex[id] = struct{}{}
	}
	t.Last = genesisLayer.Index()
	// set last evicted one layer earlier since we look for things starting one layer after last evicted
	t.LastEvicted = genesisLayer.Index() - 1
	t.Verified = genesisLayer.Index()
}

// evict makes sure we only keep a window of the last hdist layers.
func (t *turtle) evict(ctx context.Context) {
	logger := t.logger.WithContext(ctx)

	// Don't evict before we've verified at least hdist layers
	if t.Verified <= types.GetEffectiveGenesis()+t.Hdist {
		return
	}

	// TODO: fix potential leak when we can't verify but keep receiving layers

	// prevent overflow
	if t.Verified < t.WindowSize {
		return
	}
	windowStart := t.Verified - t.WindowSize
	if windowStart <= t.LastEvicted {
		return
	}
	logger.With().Info("tortoise window start", windowStart)

	// evict from last evicted to the beginning of our window
	for layerToEvict := t.LastEvicted + 1; layerToEvict < windowStart; layerToEvict++ {
		logger.With().Debug("evicting layer", layerToEvict)
		for blk := range t.BlockOpinionsByLayer[layerToEvict] {
			delete(t.GoodBlocksIndex, blk)
		}
		delete(t.BlockOpinionsByLayer, layerToEvict)
	}
	t.LastEvicted = windowStart - 1
}

func blockIDsToString(input []types.BlockID) string {
	str := "["
	for i, b := range input {
		str += b.String()
		if i != len(input)-1 {
			str += ","
		}
	}
	str += "]"
	return str
}

// returns the binary local opinion on the validity of a block in a layer (support or against)
// TODO: cache but somehow check for changes (e.g., late-finishing Hare), maybe check hash?
func (t *turtle) getSingleInputVectorFromDB(ctx context.Context, lyrid types.LayerID, blockid types.BlockID) (vec, error) {
	if lyrid <= types.GetEffectiveGenesis() {
		return support, nil
	}

	input, err := t.bdp.GetLayerInputVectorByID(lyrid)
	if err != nil {
		return abstain, err
	}

	t.logger.WithContext(ctx).With().Debug("got input vector from db",
		lyrid,
		log.FieldNamed("query_block", blockid),
		log.String("input", blockIDsToString(input)))

	for _, bl := range input {
		if bl == blockid {
			return support, nil
		}
	}

	return against, nil
}

func (t *turtle) checkBlockAndGetInputVector(
	ctx context.Context,
	diffList []types.BlockID,
	className string,
	voteVector vec,
	baseBlockLayer types.LayerID,
) bool {
	logger := t.logger.WithContext(ctx)
	for _, exceptionBlockID := range diffList {
		if exceptionBlock, err := t.bdp.GetBlock(exceptionBlockID); err != nil {
			logger.With().Error("inconsistent state: can't find block from diff list",
				log.FieldNamed("exception_block_id", exceptionBlockID))
			return false
		} else if exceptionBlock.LayerIndex < baseBlockLayer {
			logger.With().Error("good block candidate contains exception block older than its base block",
				log.FieldNamed("older_block", exceptionBlockID),
				log.FieldNamed("older_layer", exceptionBlock.LayerIndex),
				log.FieldNamed("base_block_lyr", baseBlockLayer))
			return false
		} else if v, err := t.getSingleInputVectorFromDB(ctx, exceptionBlock.LayerIndex, exceptionBlockID); err != nil {
			logger.With().Error("unable to get single input vector for exception block",
				log.FieldNamed("older_block", exceptionBlockID),
				log.FieldNamed("older_layer", exceptionBlock.LayerIndex),
				log.FieldNamed("base_block_lyr", baseBlockLayer),
				log.Err(err))
			return false
		} else if v != voteVector {
			logger.With().Debug("not adding block to good blocks because its vote differs from input vector",
				log.FieldNamed("older_block", exceptionBlock.ID()),
				log.FieldNamed("older_layer", exceptionBlock.LayerIndex),
				log.FieldNamed("input_vote", v),
				log.String("block_vote", className))
			return false
		}
	}

	return true
}

// convert two vectors, of (1) raw candidate block IDs for a layer and (2) an opinion vector of blocks we believe belong
// in the layer, into a map of votes for each of these blocks
func (t *turtle) voteVectorForLayer(
	candidateBlocks []types.BlockID, opinionVec []types.BlockID) (voteMap map[types.BlockID]vec) {
	voteMap = make(map[types.BlockID]vec, len(candidateBlocks))
	if opinionVec == nil {
		// nil means abstain, i.e., we have no opinion on blocks in this layer
		for _, b := range candidateBlocks {
			voteMap[b] = abstain
		}
		return
	}

	// add support for all blocks in input vector
	for _, b := range opinionVec {
		voteMap[b] = support
	}

	// vote against all layer blocks not in input vector
	for _, b := range candidateBlocks {
		if _, ok := voteMap[b]; !ok {
			voteMap[b] = against
		}
	}
	return
}

// BaseBlock finds and returns a good block from a recent layer that's suitable to serve as a base block for opinions
// for newly-constructed blocks. Also includes vectors of exceptions, where the current local opinion differs from
// (or is newer than) that of the base block.
func (t *turtle) BaseBlock(ctx context.Context) (types.BlockID, [][]types.BlockID, error) {
	logger := t.logger.WithContext(ctx)

	// look at good blocks backwards from most recent processed layer to find a suitable base block
	// TODO: optimize by, e.g., trying to minimize the size of the exception list (rather than first match)
	// see https://github.com/spacemeshos/go-spacemesh/issues/2402
	for layerID := t.Last; layerID > t.LastEvicted; layerID-- {
		for block, opinion := range t.BlockOpinionsByLayer[layerID] {
			if _, ok := t.GoodBlocksIndex[block]; !ok {
				logger.With().Debug("not considering block not marked good as base block candidate",
					log.FieldNamed("last_layer", t.Last),
					layerID,
					block)
				continue
			}

			// Calculate the set of exceptions between the base block opinion and latest local opinion
			logger.With().Debug("found candidate base block", block, layerID)
			exceptionVectorMap, err := t.calculateExceptions(ctx, layerID, opinion)
			if err != nil {
				logger.With().Warning("error calculating vote exceptions for block",
					log.FieldNamed("last_layer", t.Last),
					layerID,
					block,
					log.Err(err))
				continue
			}

			logger.With().Info("chose baseblock",
				log.FieldNamed("last_layer", t.Last),
				log.FieldNamed("base_block_layer", layerID),
				block,
				log.Int("against_count", len(exceptionVectorMap[0])),
				log.Int("support_count", len(exceptionVectorMap[1])),
				log.Int("neutral_count", len(exceptionVectorMap[2])))

			return block, [][]types.BlockID{
				blockMapToArray(exceptionVectorMap[0]),
				blockMapToArray(exceptionVectorMap[1]),
				blockMapToArray(exceptionVectorMap[2]),
			}, nil
		}
	}

	// TODO: special error encoding when exceeding exception list size
	return types.BlockID{0}, nil, errNoBaseBlockFound
}

// calculate and return a list of exceptions, i.e., differences between the opinions of a base block and the local
// opinion
func (t *turtle) calculateExceptions(
	ctx context.Context,
	blockLayerID types.LayerID,
	baseBlockOpinion Opinion, // candidate base block's opinion vector
) ([]map[types.BlockID]struct{}, error) {
	logger := t.logger.WithContext(ctx).WithFields(log.FieldNamed("base_block_layer_id", blockLayerID))

	// using maps prevents duplicates
	againstDiff := make(map[types.BlockID]struct{})
	forDiff := make(map[types.BlockID]struct{})
	neutralDiff := make(map[types.BlockID]struct{})

	// we support all genesis blocks by default
	if blockLayerID == types.GetEffectiveGenesis() {
		for _, i := range types.BlockIDs(mesh.GenesisLayer().Blocks()) {
			forDiff[i] = struct{}{}
		}
	}

	// Add latest layers input vector results to the diff
	// Note: a block may only be selected as a candidate base block if it's marked "good", and it may only be marked
	// "good" if its own base block is marked "good" and all exceptions it contains agree with our local opinion.
	// We only look for and store exceptions within the sliding window set of layers as an optimization, but a block
	// can contain exceptions from any layer, back to genesis.
	startLayer := t.LastEvicted + 1
	if startLayer < types.GetEffectiveGenesis() {
		startLayer = types.GetEffectiveGenesis()
	}
	for layerID := startLayer; layerID <= t.Last; layerID++ {
		logger := logger.WithFields(log.FieldNamed("diff_layer_id", layerID))
		logger.Debug("checking input vector diffs")

		layerBlockIds, err := t.bdp.LayerBlockIds(layerID)
		if err != nil {
			if err != leveldb.ErrClosed {
				// this should not happen! we only look at layers up to the last processed layer, and we only process
				// layers with valid block data.
				logger.With().Error("no block ids for layer in database", log.Err(err))
			}
			return nil, err
		}

		// helper function for adding diffs
		addDiffs := func(bid types.BlockID, voteClass string, voteVec vec, diffMap map[types.BlockID]struct{}) {
			if v, ok := baseBlockOpinion.BlockOpinions[bid]; !ok || v != voteVec {
				logger.With().Debug("added vote diff",
					log.FieldNamed("diff_block", bid),
					log.String("diff_class", voteClass))
				diffMap[bid] = struct{}{}
			}
		}

		// get local opinion for layer
		layerInputVector, err := t.layerOpinionVector(ctx, layerID)
		if err != nil {
			// an error here signifies a real database failure
			logger.With().Error("unable to calculate local opinion for layer", log.Err(err))
			return nil, err
		}

		// otherwise, nil means we should abstain
		if layerInputVector == nil {
			// still waiting for Hare results, vote neutral and move on
			logger.With().Debug("input vector is empty, adding neutral diffs", log.Err(err))
			for _, b := range layerBlockIds {
				addDiffs(b, "neutral", abstain, neutralDiff)
			}
			continue
		}
		logger.With().Debug("got local opinion vector for layer", log.Int("count", len(layerInputVector)))

		inInputVector := make(map[types.BlockID]struct{})

		// Add diffs FOR blocks that are in the input vector, but where the base block has no opinion or does not
		// explicitly support the block
		for _, b := range layerInputVector {
			inInputVector[b] = struct{}{}
			addDiffs(b, "support", support, forDiff)
		}

		// Next, we need to make sure we vote AGAINST all blocks in the layer that are not in the input vector (and
		// where we are no longer waiting for Hare results, see above)
		for _, b := range layerBlockIds {
			if _, ok := inInputVector[b]; !ok {
				addDiffs(b, "against", against, againstDiff)
			}
		}

		// Finally, we need to consider the case where the base block supports a block in this layer that is not in our
		// input vector (e.g., one we haven't seen), by adding a diff against the block
		// TODO: this is not currently possible since base block opinions aren't indexed by layer. See
		// https://github.com/spacemeshos/go-spacemesh/issues/2424
		//for b, v := range baseBlockOpinion.BlockOpinions {
		//	if _, ok := inInputVector[b]; !ok && v != against {
		//		addDiffs(b, "against", against, againstDiff)
		//	}
		//}
	}

	// check if exceeded max no. exceptions
	explen := len(againstDiff) + len(forDiff) + len(neutralDiff)
	if explen > t.MaxExceptions {
		return nil, fmt.Errorf("%s (%v)", errstrTooManyExceptions, explen)
	}

	return []map[types.BlockID]struct{}{againstDiff, forDiff, neutralDiff}, nil
}

// BlockWeight returns the weight to assign to one block's vote for another.
// Note: weight depends on more than just the weight of the voting block. It also depends on contextual factors such as
// whether or not the block's ATX was received on time, and on how old the layer is.
// TODO: for now it's probably sufficient to adjust weight based on whether the ATX was received on time, or late, for
//   the current epoch. Assign weight of zero to late ATXs for the current epoch?
func (t *turtle) BlockWeight(votingBlock, blockVotedOn types.BlockID) int {
	return 1
}

// Persist saves the current tortoise state to the database
func (t *turtle) persist() error {
	return t.bdp.Persist(mesh.TORTOISE, t)
}

// RecoverVerifyingTortoise retrieves the latest saved tortoise from the database
func RecoverVerifyingTortoise(mdb retriever) (interface{}, error) {
	return mdb.Retrieve(mesh.TORTOISE, &turtle{})
}

func (t *turtle) processBlock(ctx context.Context, block *types.Block) error {
	logger := t.logger.WithContext(ctx).WithFields(
		log.FieldNamed("processing_block_id", block.ID()),
		log.FieldNamed("processing_block_layer", block.LayerIndex))

	// When a new block arrives, we look up the block it points to in our table,
	// and add the corresponding vector (multiplied by the block weight) to our own vote-totals vector.
	// We then add the vote difference vector and the explicit vote vector to our vote-totals vector.
	logger.With().Debug("processing block", block.Fields()...)
	logger.With().Debug("getting baseblock", log.FieldNamed("base_block_id", block.BaseBlock))

	baseBlock, err := t.bdp.GetBlock(block.BaseBlock)
	if err != nil {
		return errBaseBlockNotInDatabase
	}

	logger.With().Debug("block supports", types.BlockIdsField(block.BlockHeader.ForDiff))
	logger.With().Debug("checking baseblock", baseBlock.Fields()...)

	layerOpinions, ok := t.BlockOpinionsByLayer[baseBlock.LayerIndex]
	if !ok {
		return fmt.Errorf("%s: %v, %v", errstrBaseBlockLayerMissing, block.BaseBlock, baseBlock.LayerIndex)
	}

	baseBlockOpinion, ok := layerOpinions[baseBlock.ID()]
	if !ok {
		return fmt.Errorf("%s: %v, %v", errstrBaseBlockNotFoundInLayer, block.BaseBlock, baseBlock.LayerIndex)
	}

	// TODO: this logic would be simpler if For and Against were a single list
	// TODO: save and vote against blocks that exceed the max exception list size (DoS prevention)
	opinion := make(map[types.BlockID]vec)
	for _, b := range block.ForDiff {
		opinion[b] = support.Multiply(t.BlockWeight(block.ID(), b))
	}
	for _, b := range block.AgainstDiff {
		// this could only happen in malicious blocks, and they should not pass a syntax check, but check here just
		// to be extra safe
		if _, alreadyVoted := opinion[b]; alreadyVoted {
			return fmt.Errorf("%s %v", errstrConflictingVotes, block.ID())
		}
		opinion[b] = against.Multiply(t.BlockWeight(block.ID(), b))
	}
	for _, b := range block.NeutralDiff {
		if _, alreadyVoted := opinion[b]; alreadyVoted {
			return fmt.Errorf("%s %v", errstrConflictingVotes, block.ID())
		}
		opinion[b] = abstain
	}
	for blk, vote := range baseBlockOpinion.BlockOpinions {
		// add base block vote only if there were no exceptions
		if _, exists := opinion[blk]; !exists {
			opinion[blk] = vote
		}
	}

	logger.With().Debug("adding block to block opinions table")
	t.BlockOpinionsByLayer[block.LayerIndex][block.ID()] = Opinion{BlockOpinions: opinion}
	return nil
}

// ProcessNewBlocks processes the votes of a set of blocks, records their opinions, and marks good blocks good.
// The blocks do not all have to be in the same layer, but if they span multiple layers, they must be sorted by LayerID.
func (t *turtle) ProcessNewBlocks(ctx context.Context, blocks []*types.Block) error {
	if len(blocks) == 0 {
		// nothing to do
		t.logger.WithContext(ctx).Warning("cannot process empty block list")
		return nil
	}
	if err := t.processBlocks(ctx, blocks); err != nil {
		return err
	}

	// attempt to verify layers up to the latest one for which we have new block data
	return t.verifyLayers(ctx)
}

func (t *turtle) processBlocks(ctx context.Context, blocks []*types.Block) error {
	logger := t.logger.WithContext(ctx)
	lastLayerID := types.LayerID(0)

	logger.With().Info("tortoise handling incoming block data", log.Int("num_blocks", len(blocks)))

	// process the votes in all layer blocks and update tables
	for _, b := range blocks {
		if b.LayerIndex < lastLayerID {
			return errNotSorted
		} else if b.LayerIndex > lastLayerID {
			lastLayerID = b.LayerIndex
		}
		if _, ok := t.BlockOpinionsByLayer[b.LayerIndex]; !ok {
			t.BlockOpinionsByLayer[b.LayerIndex] = make(map[types.BlockID]Opinion, t.AvgLayerSize)
		}
		if err := t.processBlock(ctx, b); err != nil {
			logger.With().Error("error processing block", b.ID(), log.Err(err))
		}
	}

	numGood := 0

	// Go over all blocks, in order. Mark block i "good" if:
	for _, b := range blocks {
		logger := logger.WithFields(b.ID(), log.FieldNamed("base_block_id", b.BaseBlock))
		// (1) the base block is marked as good
		if _, good := t.GoodBlocksIndex[b.BaseBlock]; !good {
			logger.Debug("not marking block as good because baseblock is not good")
		} else if baseBlock, err := t.bdp.GetBlock(b.BaseBlock); err != nil {
			logger.With().Error("inconsistent state: base block not found", log.Err(err))
		} else if true &&
			// (2) all diffs appear after the base block and are consistent with the input vote vector
			t.checkBlockAndGetInputVector(ctx, b.ForDiff, "support", support, baseBlock.LayerIndex) &&
			t.checkBlockAndGetInputVector(ctx, b.AgainstDiff, "against", against, baseBlock.LayerIndex) &&
			t.checkBlockAndGetInputVector(ctx, b.NeutralDiff, "abstain", abstain, baseBlock.LayerIndex) {
			logger.Debug("marking block good")
			t.GoodBlocksIndex[b.ID()] = struct{}{}
			numGood++
		} else {
			logger.Debug("not marking block good")
		}
	}

	logger.With().Info("finished marking good blocks",
		log.Int("total_blocks", len(blocks)),
		log.Int("good_blocks", numGood))

	if t.Last < lastLayerID {
		logger.With().Warning("got blocks for new layer before receiving layer, updating highest layer seen",
			log.FieldNamed("previous_highest", t.Last),
			log.FieldNamed("new_highest", lastLayerID))
		t.Last = lastLayerID
	}

	return nil
}

// HandleIncomingLayer processes all layer block votes
// returns the old pbase and new pbase after taking into account block votes
func (t *turtle) HandleIncomingLayer(ctx context.Context, layerID types.LayerID) error {
	if err := t.handleLayerBlocks(ctx, layerID); err != nil {
		return err
	}

	// attempt to verify layers up to the latest one for which we have new block data
	return t.verifyLayers(ctx)
}

func (t *turtle) handleLayerBlocks(ctx context.Context, layerID types.LayerID) error {
	logger := t.logger.WithContext(ctx).WithFields(layerID)

	if layerID <= types.GetEffectiveGenesis() {
		logger.Debug("not attempting to handle genesis layer")
		return nil
	}

	// Note: we don't compare newlyr and t.Verified, so this method could be called again on an already-verified layer.
	// That would update the stored block opinions but it would not attempt to re-verify an already-verified layer.

	// read layer blocks
	layerBlocks, err := t.bdp.LayerBlocks(layerID)
	if err != nil {
		return fmt.Errorf("unable to read contents of layer %v: %w", layerID, err)
	}
	if len(layerBlocks) == 0 {
		// nothing to do
		t.logger.WithContext(ctx).Warning("cannot process empty layer block list")
		return nil
	}

	if t.Last < layerID {
		t.Last = layerID
	}

	logger.With().Info("tortoise handling incoming layer", log.Int("num_blocks", len(layerBlocks)))
	return t.processBlocks(ctx, layerBlocks)
}

// loops over all layers from the last verified up to a new target layer and attempts to verify each in turn
func (t *turtle) verifyLayers(ctx context.Context) error {
	logger := t.logger.WithContext(ctx).WithFields(
		log.FieldNamed("verification_target", t.Last),
		log.FieldNamed("old_verified", t.Verified))
	logger.Info("starting layer verification")

	// we perform eviction here because it should happen after the verified layer advances
	defer t.evict(ctx)

	// attempt to verify each layer from the last verified up to one prior to the newly-arrived layer.
	// this is the full range of unverified layers that we might possibly be able to verify at this point.
	// Note: t.Verified is initialized to the effective genesis layer, so the first candidate layer here necessarily
	// follows and is post-genesis. There's no need for an additional check here.
candidateLayerLoop:
	for candidateLayerID := t.Verified + 1; candidateLayerID < t.Last; candidateLayerID++ {
		logger := logger.WithFields(log.FieldNamed("candidate_layer", candidateLayerID))

		// it's possible that self-healing already validated a layer
		if t.Verified >= candidateLayerID {
			logger.Info("self-healing already validated this layer")
			continue
		}

		logger.Info("attempting to verify candidate layer")

		// note: if the following checks fail, we just return rather than trying to verify later layers.
		// we don't presently support verifying layer N+1 when layer N hasn't been verified.

		layerBlockIds, err := t.bdp.LayerBlockIds(candidateLayerID)
		if err != nil {
			return fmt.Errorf("%s %v: %w", errstrCantFindLayer, candidateLayerID, err)
		}

		// get the local opinion for this layer. below, we calculate the global opinion on each block in the layer and
		// check if it agrees with this local opinion.
		rawLayerInputVector, err := t.layerOpinionVector(ctx, candidateLayerID)
		if err != nil {
			// an error here signifies a real database failure
			return fmt.Errorf("%s %v: %w", errstrUnableToCalculateLocalOpinion, candidateLayerID, err)
		}

		// otherwise, nil means we should abstain
		if rawLayerInputVector == nil {
			logger.With().Warning("input vector abstains on all blocks in layer", candidateLayerID)
		}
		localLayerOpinionVec := t.voteVectorForLayer(layerBlockIds, rawLayerInputVector)
		if len(localLayerOpinionVec) == 0 {
			// warn about this to be safe, but we do allow empty layers and must be able to verify them
			logger.With().Warning("empty vote vector for layer", candidateLayerID)
		}

		contextualValidity := make(map[types.BlockID]bool, len(layerBlockIds))

		// Count the votes of good blocks. localOpinionOnBlock is our local opinion on this block.
		// Declare the vote vector "verified" up to position k if the total weight exceeds the confidence threshold in
		// all positions up to k: in other words, we can verify a layer k if the total weight of the global opinion
		// exceeds the confidence threshold, and agrees with local opinion.
		for blockID, localOpinionOnBlock := range localLayerOpinionVec {
			// count the votes of the input vote vector by summing the voting weight of good blocks
			logger.With().Debug("summing votes for candidate layer block",
				blockID,
				log.FieldNamed("layer_start", candidateLayerID+1),
				log.FieldNamed("layer_end", t.Last))
			sum := t.sumVotesForBlock(ctx, blockID, candidateLayerID+1, func(votingBlockID types.BlockID) bool {
				if _, isgood := t.GoodBlocksIndex[votingBlockID]; !isgood {
					logger.With().Debug("not counting vote of block not marked good",
						log.FieldNamed("voting_block", votingBlockID))
					return false
				}
				return true
			})

			// check that the total weight exceeds the global threshold
			globalOpinionOnBlock := calculateOpinionWithThreshold(
				t.logger, sum, t.AvgLayerSize, t.GlobalThreshold, float64(t.Last-candidateLayerID))
			logger.With().Debug("verifying tortoise calculated global opinion on block",
				log.FieldNamed("block_voted_on", blockID),
				candidateLayerID,
				log.FieldNamed("global_vote_sum", sum),
				log.FieldNamed("global_opinion", globalOpinionOnBlock),
				log.FieldNamed("local_opinion", localOpinionOnBlock))

			// At this point, we have all of the data we need to make a decision on this block. There are three possible
			// outcomes:
			// 1. record our opinion on this block and go on evaluating the rest of the blocks in this layer to see if
			//    we can verify the layer (if local and global consensus match, and global consensus is decided)
			// 2. keep waiting to verify the layer (if not, and the layer is relatively recent)
			// 3. trigger self-healing (if not, and the layer is sufficiently old)
			consensusMatches := globalOpinionOnBlock == localOpinionOnBlock
			globalOpinionDecided := globalOpinionOnBlock != abstain

			if consensusMatches && globalOpinionDecided {
				// Opinion on this block is decided, save and keep going
				contextualValidity[blockID] = globalOpinionOnBlock == support
				continue
			}

			// If, for any block in this layer, the global opinion (summed block votes) disagrees with our vote (the
			// input vector), or if the global opinion is abstain, then we do not verify this layer. This could be the
			// result of a reorg (e.g., resolution of a network partition), or a malicious peer during sync, or
			// disagreement about Hare success.
			if !consensusMatches {
				logger.With().Warning("global opinion on block differs from our vote, cannot verify layer",
					blockID,
					log.FieldNamed("global_opinion", globalOpinionOnBlock),
					log.FieldNamed("local_opinion", localOpinionOnBlock))
			}

			// There are only two scenarios that could result in a global opinion of abstain: if everyone is still
			// waiting for Hare to finish for a layer (i.e., it has not yet succeeded or failed), or a balancing attack.
			// The former is temporary and will go away after `zdist' layers. And it should be true of an entire layer,
			// not just of a single block. The latter could cause the global opinion of a single block to permanently
			// be abstain. As long as the contextual validity of any block in a layer is unresolved, we cannot verify
			// the layer (since the effectiveness of each transaction in the layer depends upon the contents of the
			// entire layer and transaction ordering). Therefore we have to enter self-healing in this case.
			// TODO: abstain only for entire layer at a time, not for individual blocks (optimization)
			if !globalOpinionDecided {
				logger.With().Warning("global opinion on block is abstain, cannot verify layer",
					blockID,
					log.FieldNamed("global_opinion", globalOpinionOnBlock),
					log.FieldNamed("local_opinion", localOpinionOnBlock))
			}

			// Verifying tortoise will wait `zdist' layers for consensus, then an additional `ConfidenceParam'
			// layers until all other nodes achieve consensus. If it's still stuck after this point, i.e., if the gap
			// between this unverified candidate layer and the latest layer is greater than this distance, then we trigger
			// self-healing. But there's no point in trying to heal a layer that's not at least Hdist layers old since
			// we only consider the local opinion for recent layers.
			if candidateLayerID > t.Last {
				logger.With().Panic("candidate layer is higher than last layer received",
					log.FieldNamed("last_layer", t.Last))
			}
			logger.With().Debug("considering attempting to heal layer",
				log.FieldNamed("layer_cutoff", t.layerCutoff()),
				log.FieldNamed("zdist", t.Zdist),
				log.FieldNamed("last_layer_received", t.Last),
				log.FieldNamed("confidence_param", t.ConfidenceParam))
			if candidateLayerID < t.layerCutoff() && t.Last-candidateLayerID > t.Zdist+t.ConfidenceParam {
				lastLayer := t.Last
				// don't attempt to heal layers newer than Hdist
				if lastLayer > t.layerCutoff() {
					lastLayer = t.layerCutoff()
				}
				lastVerified := t.Verified
				t.selfHealing(ctx, lastLayer)

				// if self-healing made progress, short-circuit processing of this layer, but allow verification of
				// later layers to continue
				if t.Verified > lastVerified {
					continue candidateLayerLoop
				}
				// otherwise, if self-healing didn't make any progress, there's no point in continuing to attempt
				// verification
			}

			// give up trying to verify layers and keep waiting
			// TODO: continue to verify later layers, even after failing to verify a layer.
			//   See https://github.com/spacemeshos/go-spacemesh/issues/2403.
			logger.With().Info("failed to verify candidate layer, will reattempt later")
			return nil
		}

		// Declare the vote vector "verified" up to this layer and record the contextual validity for all blocks in this
		// layer
		for blk, v := range contextualValidity {
			if err := t.bdp.SaveContextualValidity(blk, candidateLayerID, v); err != nil {
				logger.With().Error("error saving contextual validity on block", blk, log.Err(err))
			}
		}
		t.Verified = candidateLayerID
		logger.With().Info("verified candidate layer")
	}

	return nil
}

// for layers older than this point, we vote according to global opinion (rather than local opinion)
func (t *turtle) layerCutoff() types.LayerID {
	// if we haven't seen at least Hdist layers yet, we always rely on local opinion
	if t.Last < t.Hdist {
		return 0
	}
	return t.Last - t.Hdist
}

// return the set of blocks we currently consider valid for the layer. factors in both local and global opinion,
// depending how old the layer is, and also uses weak coin to break ties.
func (t *turtle) layerOpinionVector(ctx context.Context, layerID types.LayerID) ([]types.BlockID, error) {
	logger := t.logger.WithContext(ctx).WithFields(layerID)
	var voteAbstain, voteAgainstAll []types.BlockID // nil slice, by default
	voteAgainstAll = make([]types.BlockID, 0, 0)

	// for layers older than hdist, we vote according to global opinion
	if layerID < t.layerCutoff() {
		if layerID > t.Verified {
			// this layer has not yet been verified
			// we must have an opinion about older layers at this point. if the layer hasn't been verified yet, count votes
			// and see if they pass the local threshold. if not, use the current weak coin instead to determine our vote for
			// the blocks in the layer.
			// TODO: do we need to/can we somehow cache this?
			layerBlockIds, err := t.bdp.LayerBlockIds(layerID)
			logger.With().Info("counting votes for and against blocks in old, unverified layer",
				log.Int("num_blocks", len(layerBlockIds)))
			if err != nil {
				return nil, err
			}
			layerBlocks := make(map[types.BlockID]struct{}, len(layerBlockIds))
			for _, blockID := range layerBlockIds {
				logger := logger.WithFields(log.FieldNamed("candidate_block_id", blockID))
				sum := t.sumVotesForBlock(ctx, blockID, layerID+1, func(id types.BlockID) bool { return true })
				localOpinionOnBlock := calculateOpinionWithThreshold(t.logger, sum, t.AvgLayerSize, t.LocalThreshold, 1)
				logger.With().Debug("local opinion on block in old layer",
					sum,
					log.FieldNamed("local_opinion", localOpinionOnBlock))
				if localOpinionOnBlock == support {
					layerBlocks[blockID] = struct{}{}
				} else if localOpinionOnBlock == abstain {
					// abstain means the votes for and against this block did not cross the local threshold.
					// if any block in this layer doesn't cross the local threshold, rescore the entire layer using the
					// weak coin. note: we use the weak coin for the previous layer since we expect to receive blocks
					// for a layer before hare finishes for that layer, i.e., before the weak coin value is ready for
					// the layer.
					if coin, exists := t.bdp.GetCoinflip(ctx, t.Last-1); exists {
						logger.With().Info("rescoring all blocks in old layer using weak coin",
							log.Int("count", len(layerBlockIds)),
							log.Bool("coinflip", coin),
							log.FieldNamed("coinflip_layer", t.Last-1))
						if coin {
							// heads on the weak coin means vote for all blocks in the layer
							return layerBlockIds, nil
						}
						// tails on the weak coin means vote against all blocks in the layer
						return voteAgainstAll, nil
					}
					return nil, fmt.Errorf("%s %v", errstrNoCoinflip, t.Last)
				} // (nothing to do if local opinion is against, just don't include block in output)
			}
			logger.With().Debug("local opinion supports blocks in old, unverified layer",
				log.Int("count", len(layerBlocks)))
			return blockMapToArray(layerBlocks), nil
		}
		// this layer has been verified, so we should be able to read the set of contextual blocks
		logger.Debug("using contextually valid blocks as opinion on old, verified layer")
		layerBlocks, err := t.bdp.ContextuallyValidBlock(layerID)
		if err != nil {
			return nil, err
		}
		logger.With().Debug("got contextually valid blocks for layer",
			log.Int("count", len(layerBlocks)))
		return blockMapToArray(layerBlocks), nil
	}

	// for newer layers, we vote according to the local opinion (input vector, from hare or sync)
	opinionVec, err := t.bdp.GetLayerInputVectorByID(layerID)
	if err != nil {
		if errors.Is(err, mesh.ErrInvalidLayer) {
			// Hare already failed for this layer, so we want to vote against all blocks in the layer. Just return an
			// empty list.
			logger.Debug("local opinion is against all blocks in layer where hare failed")
			return voteAgainstAll, nil
		} else if t.Last > t.Zdist && layerID < t.Last-t.Zdist {
			// Layer has passed the Hare abort distance threshold, so we give up waiting for Hare results. At this point
			// our opinion on this layer is that we vote against blocks (i.e., we support an empty layer).
			logger.With().Debug("local opinion on layer beyond hare abort window is against all blocks",
				log.Err(err))
			return voteAgainstAll, nil
		} else {
			// Hare hasn't failed and layer has not passed the Hare abort threshold, so we abstain while we keep waiting
			// for Hare results.
			logger.With().Warning("local opinion abstains on all blocks in layer", log.Err(err))
			return voteAbstain, nil
		}
	}
	logger.With().Debug("got input vector for layer", log.Int("count", len(opinionVec)))
	return opinionVec, nil
}

func (t *turtle) sumVotesForBlock(
	ctx context.Context,
	blockID types.BlockID, // the block we're summing votes for/against
	startLayer types.LayerID,
	filter func(types.BlockID) bool,
) (sum vec) {
	sum = abstain
	logger := t.logger.WithContext(ctx).WithFields(
		log.FieldNamed("start_layer", startLayer),
		log.FieldNamed("end_layer", t.Last),
		log.FieldNamed("block_voting_on", blockID),
		log.FieldNamed("layer_voting_on", startLayer-1))
	// TODO LANE: need to factor in theta
	for voteLayer := startLayer; voteLayer <= t.Last; voteLayer++ {
		logger := logger.WithFields(voteLayer)
		logger.With().Debug("summing layer votes",
			log.Int("count", len(t.BlockOpinionsByLayer[voteLayer])))
		for votingBlockID, votingBlockOpinion := range t.BlockOpinionsByLayer[voteLayer] {
			logger := logger.WithFields(log.FieldNamed("voting_block", votingBlockID))
			if !filter(votingBlockID) {
				logger.Debug("voting block did not pass filter, not counting its vote")
				continue
			}

			// check if this block has an opinion on the block to vote on
			// no opinion (on a block in an older layer) counts as an explicit vote against the block
			if opinionVote, exists := votingBlockOpinion.BlockOpinions[blockID]; exists {
				logger.With().Debug("adding block opinion to vote sum",
					log.FieldNamed("vote", opinionVote),
					sum)
				sum = sum.Add(opinionVote.Multiply(t.BlockWeight(votingBlockID, blockID)))
			} else {
				logger.Debug("no opinion on older block, counting vote against")
				sum = sum.Add(against.Multiply(t.BlockWeight(votingBlockID, blockID)))
			}
		}
	}
	return
}

// Manually count all votes for all layers since the last verified layer, up to the newly-arrived layer (there's no
// point in going further since we have no new information about any newer layers). Self-healing does not take into
// consideration local opinion, it relies solely on global opinion.
func (t *turtle) selfHealing(ctx context.Context, targetLayerID types.LayerID) {
	// These are our starting values
	pbaseOld := t.Verified
	pbaseNew := t.Verified

	// TODO: optimize this algorithm using, e.g., a triangular matrix rather than nested loops

	for candidateLayerID := pbaseOld + 1; candidateLayerID < targetLayerID; candidateLayerID++ {
		logger := t.logger.WithContext(ctx).WithFields(
			log.FieldNamed("old_verified_layer", pbaseOld),
			log.FieldNamed("new_verified_layer", pbaseNew),
			log.FieldNamed("target_layer", targetLayerID),
			log.FieldNamed("candidate_layer", candidateLayerID),
			log.FieldNamed("last_layer_received", t.Last),
			log.FieldNamed("hdist", t.Hdist))

		// we should never run on layers newer than Hdist back (from last layer received)
		// when bootstrapping, don't attempt any verification at all
		latestLayerWeCanVerify := t.Last - t.Hdist
		if t.Last < t.Hdist {
			latestLayerWeCanVerify = mesh.GenesisLayer().Index()
		}
		if candidateLayerID > latestLayerWeCanVerify {
			logger.With().Error("cannot heal layer that's not at least hdist layers old",
				log.FieldNamed("highest_healable_layer", latestLayerWeCanVerify))
			return
		}

		// Calculate the global opinion on all blocks in the layer
		// Note: we look at ALL blocks we've seen for the layer, not just those we've previously marked contextually valid
		logger.Info("self-healing attempting to verify candidate layer")

		layerBlockIds, err := t.bdp.LayerBlockIds(candidateLayerID)
		if err != nil {
			logger.Error("inconsistent state: can't find layer in database, cannot heal")

			// there's no point in trying to verify later layers so just give up now
			return
		}

		// This map keeps track of the contextual validity of all blocks in this layer
		contextualValidity := make(map[types.BlockID]bool, len(layerBlockIds))
		for _, blockID := range layerBlockIds {
			logger := logger.WithFields(log.FieldNamed("candidate_block_id", blockID))

			// count all votes for or against this block by all blocks in later layers: don't filter out any
			sum := t.sumVotesForBlock(ctx, blockID, candidateLayerID+1, func(id types.BlockID) bool { return true })

			// check that the total weight exceeds the global threshold
			globalOpinionOnBlock := calculateOpinionWithThreshold(t.logger, sum, t.AvgLayerSize, t.GlobalThreshold, float64(t.Last-candidateLayerID))
			logger.With().Debug("self-healing calculated global opinion on candidate block",
				log.FieldNamed("global_opinion", globalOpinionOnBlock),
				sum)

			if globalOpinionOnBlock == abstain {
				logger.With().Info("self-healing failed to verify candidate layer, will reattempt later")
				return
			}

			contextualValidity[blockID] = globalOpinionOnBlock == support
		}

		// TODO: do we overwrite the layer input vector in the database here?
		// TODO: do we mark approved blocks good? do we update other blocks' opinions of them?

		// record the contextual validity for all blocks in this layer
		for blk, v := range contextualValidity {
			if err := t.bdp.SaveContextualValidity(blk, candidateLayerID, v); err != nil {
				logger.With().Error("error saving contextual validity on block", blk, log.Err(err))
			}
		}
		t.Verified = candidateLayerID
		pbaseNew = candidateLayerID
		logger.With().Info("self healing verified candidate layer")
	}

	return
}