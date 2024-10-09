go 1.22.2

toolchain go1.22.3

module github.com/cosmos/ibc-go/simapp

replace (
	github.com/cosmos/ibc-go/modules/capability => ../modules/capability
	github.com/cosmos/ibc-go/v9 => ../
)

replace github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
