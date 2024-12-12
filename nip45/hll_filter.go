package nip45

import (
	"strconv"

	"github.com/nbd-wtf/go-nostr"
)

// HyperLogLogEventPubkeyOffsetForFilter returns the deterministic pubkey offset that will be used
// when computing hyperloglogs in the context of a specific filter.
//
// It returns -1 when the filter is not eligible for hyperloglog calculation.
func HyperLogLogEventPubkeyOffsetForFilter(filter nostr.Filter) int {
	if filter.IDs != nil || filter.Since != nil || filter.Until != nil || filter.Authors != nil ||
		len(filter.Kinds) != 1 || filter.Search != "" || len(filter.Tags) != 1 {
		// obvious cases in which we won't bother to do hyperloglog stuff
		return -1
	}

	// only serve the cases explicitly defined by the NIP:
	if pTags, ok := filter.Tags["p"]; ok {
		//
		// follower counts:
		if filter.Kinds[0] == 3 && len(pTags) == 1 && len(pTags[0]) == 1 && pTags[0][0] != nil {
			// 32th nibble of "p" tag
			str := *pTags[0][0]
			p, err := strconv.ParseInt(str[32:33], 16, 64)
			if err != nil {
				return -1
			}
			return int(p + 8)
		}
	} else if eTags, ok := filter.Tags["e"]; ok {
		if len(eTags) == 1 && len(eTags[0]) > 0 && eTags[0][0] != nil {
			//
			// reaction counts:
			if filter.Kinds[0] == 7 {
				str := *eTags[0][0]
				// 32th nibble of "e" tag
				p, err := strconv.ParseInt(str[32:33], 16, 64)
				if err != nil {
					return -1
				}
				return int(p + 8)
			}
		}
	}

	// everything else is false at least for now
	return -1
}
