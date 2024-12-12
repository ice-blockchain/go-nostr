package nostr

import (
	"slices"

	"github.com/mailru/easyjson"
)

type (
	TagValues []*string

	TagMap map[string][]TagValues

	Filter struct {
		IDs     []string
		Kinds   []int
		Authors []string
		Tags    TagMap
		Since   *Timestamp
		Until   *Timestamp
		Limit   int
		Search  string

		// LimitZero is or must be set when there is a "limit":0 in the filter, and not when "limit" is just omitted
		LimitZero bool `json:"-"`
	}

	Filters []Filter
)

func (m TagMap) SetLiterals(tag string, values ...string) TagMap {
	tagValues := make(TagValues, len(values))
	for i := range values {
		tagValues[i] = &values[i]
	}

	return m.Set(tag, tagValues...)
}

func (m TagMap) Set(tag string, values ...*string) TagMap {
	m[tag] = []TagValues{values}

	return m
}

func (m TagMap) Append(tag string, values ...*string) TagMap {
	m[tag] = append(m[tag], values)

	return m
}

func (m TagMap) HasValues(tag string) bool {
	for _, values := range m[tag] {
		if !values.Empty() {
			return true
		}
	}
	return false
}

func (v TagValues) Empty() bool {
	for i := range v {
		if v[i] != nil {
			return false
		}
	}
	return true
}

func (eff Filters) String() string {
	j, _ := json.Marshal(eff)
	return string(j)
}

func (eff Filters) Match(event *Event) bool {
	for _, filter := range eff {
		if filter.Matches(event) {
			return true
		}
	}
	return false
}

func (eff Filters) MatchIgnoringTimestampConstraints(event *Event) bool {
	for _, filter := range eff {
		if filter.MatchesIgnoringTimestampConstraints(event) {
			return true
		}
	}
	return false
}

func (ef Filter) String() string {
	j, _ := easyjson.Marshal(ef)
	return string(j)
}

func (ef Filter) Matches(event *Event) bool {
	if !ef.MatchesIgnoringTimestampConstraints(event) {
		return false
	}

	if ef.Since != nil && event.CreatedAt < *ef.Since {
		return false
	}

	if ef.Until != nil && event.CreatedAt > *ef.Until {
		return false
	}

	return true
}

func (ef Filter) MatchesIgnoringTimestampConstraints(event *Event) bool {
	if event == nil {
		return false
	}

	if ef.IDs != nil && !slices.Contains(ef.IDs, event.ID) {
		return false
	}

	if ef.Kinds != nil && !slices.Contains(ef.Kinds, event.Kind) {
		return false
	}

	if ef.Authors != nil && !slices.Contains(ef.Authors, event.PubKey) {
		return false
	}

	valuesOf := func(name string) Tag {
		for _, tag := range event.Tags {
			if tag.Key() == name {
				return tag[1:] // Skip the tag name.
			}
		}
		return nil
	}

	for tag, values := range ef.Tags {
		eventTagValues := valuesOf(tag)
		if eventTagValues == nil {
			return false
		}
		for i := range values {
			for j := range values[i] {
				if values[i][j] == nil {
					continue
				}
				if j >= len(eventTagValues) || *values[i][j] != eventTagValues[j] {
					return false
				}
			}
		}
	}
	return true
}

func FilterEqual(a Filter, b Filter) bool {
	if !similar(a.Kinds, b.Kinds) {
		return false
	}

	if !similar(a.IDs, b.IDs) {
		return false
	}

	if !similar(a.Authors, b.Authors) {
		return false
	}

	if len(a.Tags) != len(b.Tags) {
		return false
	}

	for f, av := range a.Tags {
		bv, ok := b.Tags[f]
		if !ok {
			return false
		}

		ok = slices.EqualFunc(av, bv, func(a, b TagValues) bool {
			if len(a) != len(b) {
				return false
			}
			for i := range len(a) {
				if !arePointerValuesEqual(a[i], b[i]) {
					return false
				}
			}
			return true
		})
		if !ok {
			return false
		}
	}

	if !arePointerValuesEqual(a.Since, b.Since) {
		return false
	}

	if !arePointerValuesEqual(a.Until, b.Until) {
		return false
	}

	if a.Search != b.Search {
		return false
	}

	if a.LimitZero != b.LimitZero {
		return false
	}

	return true
}

func (ef Filter) Clone() Filter {
	clone := Filter{
		IDs:       slices.Clone(ef.IDs),
		Authors:   slices.Clone(ef.Authors),
		Kinds:     slices.Clone(ef.Kinds),
		Limit:     ef.Limit,
		Search:    ef.Search,
		LimitZero: ef.LimitZero,
	}

	if ef.Tags != nil {
		clone.Tags = make(TagMap, len(ef.Tags))
		for k, v := range ef.Tags {
			clone.Tags[k] = slices.Clone(v)
		}
	}

	if ef.Since != nil {
		since := *ef.Since
		clone.Since = &since
	}

	if ef.Until != nil {
		until := *ef.Until
		clone.Until = &until
	}

	return clone
}

// GetTheoreticalLimit gets the maximum number of events that a normal filter would ever return, for example, if
// there is a number of "ids" in the filter, the theoretical limit will be that number of ids.
//
// It returns -1 if there are no theoretical limits.
//
// The given .Limit present in the filter is ignored.
func GetTheoreticalLimit(filter Filter) int {
	if len(filter.IDs) > 0 {
		return len(filter.IDs)
	}

	if len(filter.Kinds) == 0 {
		return -1
	}

	if len(filter.Authors) > 0 {
		allAreReplaceable := true
		for _, kind := range filter.Kinds {
			if !IsReplaceableKind(kind) {
				allAreReplaceable = false
				break
			}
		}
		if allAreReplaceable {
			return len(filter.Authors) * len(filter.Kinds)
		}

		if len(filter.Tags["d"]) > 0 {
			allAreAddressable := true
			for _, kind := range filter.Kinds {
				if !IsAddressableKind(kind) {
					allAreAddressable = false
					break
				}
			}
			if allAreAddressable {
				var dlen int
				for _, values := range filter.Tags["d"] {
					for _, value := range values {
						if value != nil {
							dlen++
						}
					}
				}
				return len(filter.Authors) * len(filter.Kinds) * dlen
			}
		}
	}

	return -1
}
