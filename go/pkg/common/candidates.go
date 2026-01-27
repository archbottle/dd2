package common

import "sort"

// CandidateMode controls how aggressively we rely on the keyword index.
//
// - Compatibility: the index is a pure optimization; we fall back to full scan.
// - StrictIndex: best-effort performance mode; we only try index candidates.
type CandidateMode int

const (
	Compatibility CandidateMode = iota
	StrictIndex
)

// Orderable allows candidate slices to be stably sorted to preserve YAML order.
// Many parser DBs rely on "first match wins", so deterministic ordering matters.
type Orderable interface {
	Order() int
}

// SelectCandidates returns an ordered list of candidates to test for a given UA.
//
// Correctness rule:
// - In Compatibility mode, if the index yields no candidates, we fall back to full scan.
// - In StrictIndex mode, if the index yields no candidates, we return nil (no scan).
//
// Note: This function does not perform the "no match in candidates -> full scan" fallback;
// that remains in parser logic, because only the parser knows whether a match was found.
func SelectCandidates[T Pattern](all []T, idx *PatternIndex[T], ua string, mode CandidateMode) []T {
	if len(all) == 0 {
		return nil
	}
	if idx == nil {
		return all
	}

	cands := idx.FindCandidates(ua)
	if len(cands) == 0 {
		if mode == Compatibility {
			return all
		}
		return nil
	}

	// Preserve YAML order where available.
	if _, ok := any(cands[0]).(Orderable); ok {
		sort.SliceStable(cands, func(i, j int) bool {
			return any(cands[i]).(Orderable).Order() < any(cands[j]).(Orderable).Order()
		})
	}
	return cands
}
