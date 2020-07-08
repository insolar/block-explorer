package load

import (
	"log"

	"github.com/skudasov/loadgen"
)

func AttackerFromName(name string) loadgen.Attack {
	switch name {
	case "get_pulses":
		return loadgen.WithMonitor(new(GetPulsesAttack))
	case "get_jet_drop_by_id":
		return loadgen.WithMonitor(new(GetJetDropByIDAttack))
	default:
		log.Fatalf("unknown attacker type: %s", name)
		return nil
	}
}
