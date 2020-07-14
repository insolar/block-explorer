package load

import (
	"encoding/csv"

	"github.com/skudasov/loadgen"
)

func NewCSVWriter(name string) (*csv.Writer, func() error) {
	f := loadgen.CreateOrReplaceFile(name)
	return csv.NewWriter(f), f.Close
}
