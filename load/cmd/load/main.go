package main

import (
	"github.com/skudasov/loadgen"

	"github.com/insolar/block-explorer/load"
)

func main() {
	loadgen.Run(load.AttackerFromName, load.CheckFromName)
}
