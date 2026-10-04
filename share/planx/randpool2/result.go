package randpool2

type result struct {
	LinkRandomID string
	Loots        []*Loot
}

// Loot 最终生成的掉落
type Loot struct {
	ItemID uint32
	Count  uint32
}

func mergeLoot(Loots1 []*Loot, Loots2 []*Loot) []*Loot {
	for i := range Loots2 {
		l2 := Loots2[i]
		merged := false
		for _, l1 := range Loots1 {
			if l2.ItemID == l1.ItemID {
				l1.Count += l2.Count
				merged = true
				break
			}
		}
		if !merged {
			Loots1 = append(Loots1, l2)
		}
	}
	return Loots1
}
