package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
)

// Simulation operation weights constants
const (
	DefaultWeightMsgUpdateParams int = 100

	OpWeightMsgUpdateParams = "op_weight_msg_update_params" // #nosec
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsgX(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateHostMsgUpdateParams,
		),
		simulation.NewWeightedProposalMsgX(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateControllerMsgUpdateParams,
		),
	}
}

// SimulateHostMsgUpdateParams returns a MsgUpdateParams for the host module
func SimulateControllerMsgUpdateParams(ctx context.Context, _ *rand.Rand, _ []simtypes.Account, _ coreaddress.Codec) (sdk.Msg, error) {
	var signer sdk.AccAddress = address.Module("gov")
	params := types.DefaultParams()
	params.HostEnabled = false

	return &types.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}, nil
}

// SimulateControllerMsgUpdateParams returns a MsgUpdateParams for the controller module
func SimulateControllerMsgUpdateParams(_ *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")
	params := controllertypes.DefaultParams()
	params.ControllerEnabled = false

	return &controllertypes.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}
}
