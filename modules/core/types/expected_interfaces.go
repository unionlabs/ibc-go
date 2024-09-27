package types

import (
	"context"

	paramtypes "cosmossdk.io/x/params/types"
)

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx context.Context, ps paramtypes.ParamSet)
}
