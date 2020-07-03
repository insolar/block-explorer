// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package storage

// GetJetIDParents returns parents of the jet id
// "0010" -> ['' 0 00 001 0010]
func GetJetIDParents(jetID string) []string {
	length := len(jetID)
	parents := make([]string, length)

	for i := 0; i < length; i++ {
		parents[i] = jetID[:i+1]
	}
	parents = append([]string{""}, parents...)
	return parents
}
