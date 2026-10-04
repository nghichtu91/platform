package randpool

import "math/rand"

// LootData 最终产出掉落
type LootData interface {
	GetLoots(r *rand.Rand, d IRandomConfigDistribution) ([]*Loot, error)
}
