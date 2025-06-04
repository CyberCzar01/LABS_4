package regexlib

func Minimize(d *DFA) *DFA {
	if d == nil || d.Start == nil {
		return d // пустой автомат → ничего минимизировать
	}

	// --- 1. начальное разбиение  ------------------------------------------
	acc, non := make(map[*dfaState]struct{}), make(map[*dfaState]struct{})
	for _, s := range d.States {
		if s.accept {
			acc[s] = struct{}{}
		} else {
			non[s] = struct{}{}
		}
	}

	partitions := make([]map[*dfaState]struct{}, 0, 2)
	if len(acc) != 0 {
		partitions = append(partitions, acc)
	}
	if len(non) != 0 {
		partitions = append(partitions, non)
	}

	// Вместо сравнения map'ов храним ИНДЕКСЫ блоков в work-множине
	work := make([]int, len(partitions))
	for i := range work {
		work[i] = i
	}

	contains := func(set map[*dfaState]struct{}, s *dfaState) bool {
		_, ok := set[s]
		return ok
	}

	// --- 2. основной цикл ---------------------------------------------------
	for len(work) > 0 {
		idx := work[0]
		work = work[1:]
		A := partitions[idx]

		for _, c := range d.Alpha {
			// X ← предобраз A по символу c
			X := make(map[*dfaState]struct{})
			for _, s := range d.States {
				if t, ok := s.trans[c]; ok && contains(A, t) {
					X[s] = struct{}{}
				}
			}

			// refine: каждая Y ∈ P = (inter ∧ diff)
			for pIdx := 0; pIdx < len(partitions); pIdx++ {
				Y := partitions[pIdx]
				inter := make(map[*dfaState]struct{})
				diff := make(map[*dfaState]struct{})

				for s := range Y {
					if contains(X, s) {
						inter[s] = struct{}{}
					} else {
						diff[s] = struct{}{}
					}
				}
				if len(inter) == 0 || len(diff) == 0 {
					continue // не разбилось
				}

				// заменить Y на два новых блока
				partitions[pIdx] = inter
				partitions = append(partitions, diff)

				// правило Hopcroft: в work кладём МЕНЬШИЙ блок
				if len(inter) < len(diff) {
					work = append(work, pIdx)
				} else {
					work = append(work, len(partitions)-1)
				}
			}
		}
	}

	// --- 3. строим уменьшенный DFA -----------------------------------------
	// каждому старому состоянию ставим в соответствие представителя блока
	representative := make(map[*dfaState]*dfaState)
	for _, P := range partitions {
		var first *dfaState
		for s := range P {
			first = s
			break
		}
		newState := &dfaState{
			id:     len(representative),
			accept: first.accept,
			trans:  make(map[rune]*dfaState),
		}
		for s := range P {
			representative[s] = newState
		}
	}

	// переносим переходы
	for old, rep := range representative {
		for c, to := range old.trans {
			rep.trans[c] = representative[to]
		}
	}

	// собираем уникальные состояния
	uniqMap := make(map[*dfaState]struct{})
	for _, s := range representative {
		uniqMap[s] = struct{}{}
	}
	uniq := make([]*dfaState, 0, len(uniqMap))
	for s := range uniqMap {
		uniq = append(uniq, s)
	}

	return &DFA{
		Start:  representative[d.Start],
		States: uniq,
		Alpha:  d.Alpha,
	}
}
