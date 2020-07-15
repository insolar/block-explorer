// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package load

import (
	"log"

	"github.com/skudasov/loadgen"
)

func AttackerFromName(name string) loadgen.Attack {
	switch name {
	case "get_pulse":
		return loadgen.WithMonitor(new(GetPulseAttack))
	case "get_pulses":
		return loadgen.WithMonitor(new(GetPulsesAttack))
	case "get_jet_drop_by_id":
		return loadgen.WithMonitor(new(GetJetDropByIDAttack))
	case "get_jet_drops_by_pulse_number":
		return loadgen.WithMonitor(new(GetJetDropsByPulseNumberAttack))
	case "get_jet_drops_by_jet_id":
		return loadgen.WithCSVMonitor(new(GetJetDropsByJetIDAttack))
	case "get_records":
		return loadgen.WithCSVMonitor(new(GetRecordsAttack))
	case "get_lifeline":
		return loadgen.WithCSVMonitor(new(GetLifelineAttack))
	case "search":
		return loadgen.WithCSVMonitor(new(SearchAttack))
	default:
		log.Fatalf("unknown attacker type: %s", name)
		return nil
	}
}
