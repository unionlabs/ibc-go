package tendermint

import (
	"context"
	"reflect"
	"time"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	stakingtypes "cosmossdk.io/x/staking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/light"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ clienttypes.ConsensusHost = (*ConsensusHost)(nil)

// ConsensusHost implements the 02-client clienttypes.ConsensusHost interface.
type ConsensusHost struct {
	stakingKeeper StakingKeeper
}

// StakingKeeper defines an expected interface for the tendermint ConsensusHost.
type StakingKeeper interface {
	GetHistoricalInfo(ctx context.Context, height int64) (stakingtypes.HistoricalInfo, error)
	UnbondingTime(ctx context.Context) (time.Duration, error)
}

// NewConsensusHost creates and returns a new ConsensusHost for tendermint consensus.
func NewConsensusHost(stakingKeeper clienttypes.StakingKeeper) clienttypes.ConsensusHost {
	return &ConsensusHost{
		stakingKeeper: stakingKeeper,
	}
}

// GetSelfConsensusState implements the 02-client clienttypes.ConsensusHost interface.
func (c *ConsensusHost) GetSelfConsensusState(ctx context.Context, height exported.Height) (exported.ConsensusState, error) {
	selfHeight, ok := height.(clienttypes.Height)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", clienttypes.Height{}, height)
	}

	// check that height revision matches chainID revision
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/7223
	revision := clienttypes.ParseChainID(sdkCtx.ChainID())
	if revision != height.GetRevisionNumber() {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "chainID revision number does not match height revision number: expected %d, got %d", revision, height.GetRevisionNumber())
	}

	histInfo, err := c.stakingKeeper.GetHistoricalInfo(ctx, int64(selfHeight.RevisionHeight))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "height %d", selfHeight.RevisionHeight)
	}

	consensusState := &ConsensusState{
		Timestamp:          histInfo.Header.Time,
		Root:               commitmenttypes.NewMerkleRoot(histInfo.Header.GetAppHash()),
		NextValidatorsHash: histInfo.Header.NextValidatorsHash,
	}

	return consensusState, nil
}

// ValidateSelfClient implements the 02-client clienttypes.ConsensusHost interface.
func (c *ConsensusHost) ValidateSelfClient(ctx context.Context, clientState exported.ClientState) error {
	tmClient, ok := clientState.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client must be a Tendermint client, expected: %T, got: %T", &ClientState{}, tmClient)
	}

	if !tmClient.FrozenHeight.IsZero() {
		return clienttypes.ErrClientFrozen
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: https://github.com/cosmos/ibc-go/issues/7223
	if sdkCtx.ChainID() != tmClient.ChainId {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "invalid chain-id. expected: %s, got: %s",
			sdkCtx.ChainID(), tmClient.ChainId)
	}

	revision := clienttypes.ParseChainID(sdkCtx.ChainID())

	// client must be in the same revision as executing chain
	if tmClient.LatestHeight.RevisionNumber != revision {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client is not in the same revision as the chain. expected revision: %d, got: %d",
			tmClient.LatestHeight.RevisionNumber, revision)
	}

	selfHeight := clienttypes.NewHeight(revision, uint64(sdkCtx.BlockHeight()))
	if tmClient.LatestHeight.GTE(selfHeight) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client has LatestHeight %d greater than or equal to chain height %d",
			tmClient.LatestHeight, selfHeight)
	}

	expectedProofSpecs := commitmenttypes.GetSDKSpecs()
	if !reflect.DeepEqual(expectedProofSpecs, tmClient.ProofSpecs) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "client has invalid proof specs. expected: %v got: %v",
			expectedProofSpecs, tmClient.ProofSpecs)
	}

	if err := light.ValidateTrustLevel(tmClient.TrustLevel.ToTendermint()); err != nil {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "trust-level invalid: %v", err)
	}

	expectedUbdPeriod, err := c.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to retrieve unbonding period")
	}

	if expectedUbdPeriod != tmClient.UnbondingPeriod {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "invalid unbonding period. expected: %s, got: %s",
			expectedUbdPeriod, tmClient.UnbondingPeriod)
	}

	if tmClient.UnbondingPeriod < tmClient.TrustingPeriod {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "unbonding period must be greater than trusting period. unbonding period (%d) < trusting period (%d)",
			tmClient.UnbondingPeriod, tmClient.TrustingPeriod)
	}

	if len(tmClient.UpgradePath) != 0 {
		// For now, SDK IBC implementation assumes that upgrade path (if defined) is defined by SDK upgrade module
		expectedUpgradePath := []string{upgradetypes.StoreKey, upgradetypes.KeyUpgradedIBCState}
		if !reflect.DeepEqual(expectedUpgradePath, tmClient.UpgradePath) {
			return errorsmod.Wrapf(clienttypes.ErrInvalidClient, "upgrade path must be the upgrade path defined by upgrade module. expected %v, got %v",
				expectedUpgradePath, tmClient.UpgradePath)
		}
	}

	return nil
}
