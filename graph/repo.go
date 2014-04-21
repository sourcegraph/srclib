package graph

type SymbolCounts struct {
	Exported      int            `json:"exported"`
	SpecificKinds map[string]int `json:"specificKinds"`
}
