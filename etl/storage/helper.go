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
