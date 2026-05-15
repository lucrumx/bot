package arbitragebot

import "github.com/lucrumx/bot/internal/exchange"

// commonSymbols returns the intersection of symbols across every exchange's instrument map.
// Used at startup to figure out which symbols can actually be arbitraged.
func commonSymbols(instruments map[string]map[string]exchange.Instrument) map[string]struct{} {
	result := map[string]struct{}{}
	first := true

	for _, byExchange := range instruments {
		if first {
			for symbol := range byExchange {
				result[symbol] = struct{}{}
			}
			first = false
			continue
		}
		for s := range result {
			if _, ok := byExchange[s]; !ok {
				delete(result, s)
			}
		}
	}

	return result
}
