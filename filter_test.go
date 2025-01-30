package nostr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterUnmarshal(t *testing.T) {
	t.Parallel()

	raw := `{"ids": ["abc"],"#e":["zzz"],"#something":[["nothing","bab"]],"since":1644254609,"search":"test"}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	require.NoError(t, err)

	require.Equal(t, "test", f.Search)

	require.Nil(t, f.Until)
	require.NotNil(t, f.Since)
	require.Equal(t, "2022-02-07", f.Since.Time().UTC().Format("2006-01-02"))

	require.Len(t, f.Tags, 2)
	require.Contains(t, f.Tags, "something")
	require.Contains(t, f.Tags, "e")
	require.Len(t, f.Tags["something"], 1)
	require.Len(t, f.Tags["e"], 1)
	require.Equal(t, "nothing", *f.Tags["something"][0][0])
	require.Equal(t, "bab", *f.Tags["something"][0][1])
	require.Equal(t, "zzz", *f.Tags["e"][0][0])
}

func TestFilterMarshal(t *testing.T) {
	t.Parallel()

	until := Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds: []int{KindTextNote, KindRecommendServer, KindEncryptedDirectMessage},
		Tags:  TagMap{}.SetLiterals("fruit", "banana", "mango"),
		Until: &until,
	})
	require.NoError(t, err)

	expected := `{"kinds":[1,2,4],"until":12345678,"#fruit":[["banana","mango"]]}`
	require.Equal(t, expected, string(filterj))
}

func TestFilterUnmarshalWithLimitZero(t *testing.T) {
	t.Parallel()

	raw := `{"ids": ["abc"],"#e":["zzz"],"limit":0,"#something":["nothing","bab"],"since":1644254609,"search":"test"}`
	var f Filter
	err := json.Unmarshal([]byte(raw), &f)
	require.NoError(t, err)

	require.Equal(t, "test", f.Search)

	require.Nil(t, f.Until)
	require.NotNil(t, f.Since)
	require.Equal(t, "2022-02-07", f.Since.Time().UTC().Format("2006-01-02"))

	require.Len(t, f.Tags, 2)
	require.Contains(t, f.Tags, "something")
	require.Contains(t, f.Tags, "e")
	require.True(t, f.LimitZero)
}

func TestFilterMarshalWithLimitZero(t *testing.T) {
	t.Parallel()

	until := Timestamp(12345678)
	filterj, err := json.Marshal(Filter{
		Kinds:     []int{KindTextNote, KindRecommendServer, KindEncryptedDirectMessage},
		Tags:      TagMap{}.SetLiterals("fruit", "banana", "mango"),
		Until:     &until,
		LimitZero: true,
	})
	require.NoError(t, err)

	expected := `{"kinds":[1,2,4],"until":12345678,"limit":0,"#fruit":[["banana","mango"]]}`
	require.Equal(t, expected, string(filterj))
}

func TestFilterMatchingLive(t *testing.T) {
	t.Parallel()

	var filter Filter
	var event Event

	json.Unmarshal([]byte(`{"kinds":[1],"authors":["a8171781fd9e90ede3ea44ddca5d3abf828fe8eedeb0f3abb0dd3e563562e1fc","1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","ed4ca520e9929dfe9efdadf4011b53d30afd0678a09aa026927e60e7a45d9244"],"since":1677033299}`), &filter)
	json.Unmarshal([]byte(`{"id":"5a127c9c931f392f6afc7fdb74e8be01c34035314735a6b97d2cf360d13cfb94","pubkey":"1d80e5588de010d137a67c42b03717595f5f510e73e42cfc48f31bae91844d59","created_at":1677033299,"kind":1,"tags":[["t","japan"]],"content":"If you like my art,I'd appreciate a coin or two!!\nZap is welcome!! Thanks.\n\n\n#japan #bitcoin #art #bananaart\nhttps://void.cat/d/CgM1bzDgHUCtiNNwfX9ajY.webp","sig":"828497508487ca1e374f6b4f2bba7487bc09fccd5cc0d1baa82846a944f8c5766918abf5878a580f1e6615de91f5b57a32e34c42ee2747c983aaf47dbf2a0255"}`), &event)

	require.True(t, filter.Matches(&event), "live filter should match")
}

func TestFilterEquality(t *testing.T) {
	t.Parallel()

	require.True(t, FilterEqual(
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}},
	), "kinds filters should be equal")

	require.True(t, FilterEqual(
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}, Tags: TagMap{}.SetLiterals("letter", "a", "b")},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion}, Tags: TagMap{}.SetLiterals("letter", "a", "b")},
	), "kind+tags filters should be equal")

	tm := Now()
	require.True(t, FilterEqual(
		Filter{
			Kinds: []int{KindEncryptedDirectMessage, KindDeletion},
			Tags:  TagMap{}.SetLiterals("letter", "a", "b").SetLiterals("fruit", "banana"),
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
		Filter{
			Kinds: []int{KindDeletion, KindEncryptedDirectMessage},
			Tags:  TagMap{}.SetLiterals("letter", "a", "b").SetLiterals("fruit", "banana"),
			Since: &tm,
			IDs:   []string{"aaaa", "bbbb"},
		},
	), "kind+2tags+since+ids filters should be equal")

	require.False(t, FilterEqual(
		Filter{Kinds: []int{KindTextNote, KindEncryptedDirectMessage, KindDeletion}},
		Filter{Kinds: []int{KindEncryptedDirectMessage, KindDeletion, KindRepost}},
	), "kinds filters shouldn't be equal")
}

func TestFilterClone(t *testing.T) {
	t.Parallel()

	ts := Now() - 60*60
	flt := Filter{
		Kinds: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Tags:  TagMap{}.SetLiterals("letter", "a", "b").SetLiterals("fruit", "banana"),
		Since: &ts,
		IDs:   []string{"9894b4b5cb5166d23ee8899a4151cf0c66aec00bde101982a13b8e8ceb972df9"},
	}
	clone := flt.Clone()
	require.True(t, FilterEqual(flt, clone), "clone is not equal:\n %v !=\n %v", flt, clone)

	clone1 := flt.Clone()
	clone1.IDs = append(clone1.IDs, "88f0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	require.False(t, FilterEqual(flt, clone1), "modifying the clone ids should cause it to not be equal anymore")

	clone2 := flt.Clone()
	clone2.Tags.Append("letter", ptr("c"))
	require.False(t, FilterEqual(flt, clone2), "modifying the clone tag items should cause it to not be equal anymore")

	clone3 := flt.Clone()
	clone3.Tags.SetLiterals("g", "drt")
	require.False(t, FilterEqual(flt, clone3), "modifying the clone tag map should cause it to not be equal anymore")

	clone4 := flt.Clone()
	*clone4.Since++
	require.False(t, FilterEqual(flt, clone4), "modifying the clone since should cause it to not be equal anymore")
}

func TestTheoreticalLimit(t *testing.T) {
	require.Equal(t, 6, GetTheoreticalLimit(Filter{IDs: []string{"a", "b", "c", "d", "e", "f"}}))
	require.Equal(t, 9, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c"}, Kinds: []int{3, 0, 10002}}))
	require.Equal(t, 4, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d"}, Kinds: []int{10050}}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d"}}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Kinds: []int{3, 0, 10002}}))
	require.Equal(t, 24, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d", "e", "f"}, Kinds: []int{30023, 30024}, Tags: TagMap{}.SetLiterals("d", "aaa", "bbb")}))
	require.Equal(t, -1, GetTheoreticalLimit(Filter{Authors: []string{"a", "b", "c", "d", "e", "f"}, Kinds: []int{30023, 30024}}))
}

func TestFilterMatches(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		Filter Filter
		Event  Event
		Match  bool
	}{
		{
			Filter: Filter{
				Tags: TagMap{
					"e": nil,
				},
			},
			Event: Event{
				Tags: Tags{{"e", "1"}},
			},
			Match: true,
		},
		{
			Filter: Filter{
				Tags: TagMap{
					"e": []TagValues{{nil, nil}, {ptr("1")}},
				},
			},
			Event: Event{
				Tags: Tags{{"e", "1"}},
			},
			Match: true,
		},
		{
			Filter: Filter{
				Tags: TagMap{
					"e": []TagValues{{nil, ptr("2"), nil}, {ptr("1")}},
				},
			},
			Event: Event{
				Tags: Tags{{"e", "1", "2", "3"}},
			},
			Match: true,
		},
		{
			Filter: Filter{
				Tags: TagMap{
					"e": []TagValues{{nil, ptr("2"), nil}, {ptr("1")}},
				},
			},
			Event: Event{
				Tags: Tags{{"e", "0", "2", "3"}},
			},
			Match: true,
		},
		{
			Filter: Filter{
				Tags: TagMap{}.SetLiterals("e", "1", "2", "3", "4"),
			},
			Event: Event{
				Tags: Tags{{"e", "1", "2", "3"}},
			},
		},
		{
			Filter: Filter{
				Tags: TagMap{
					"x": nil,
				},
			},
			Event: Event{},
		},
		{
			Filter: Filter{
				Tags: TagMap{}.
					Append("k", ptr("1")).
					Append("k", ptr("2")),
			},
			Event: Event{
				Tags: Tags{
					{"k", "1"},
				},
			},
			Match: true,
		},
		{
			Filter: Filter{
				Tags: TagMap{}.
					Append("k", ptr("1"), ptr("2")),
			},
			Event: Event{
				Tags: Tags{
					{"k", "1"},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Filter.String(), func(t *testing.T) {
			r := c.Filter.Matches(&c.Event)
			require.Equalf(t, c.Match, r, "event: %+v", c.Event)
		})
	}
}
func TestTagMapAll(t *testing.T) {
	t.Parallel()

	tagMap := TagMap{}.
		SetLiterals("fruit", "apple", "banana").
		Append("fruit", ptr("orange")).
		SetLiterals("color", "red", "yellow").
		Append("color", nil, ptr("blue"))

	require.ElementsMatch(t, []string{"apple", "banana", "orange"}, tagMap.All("fruit"))
	require.ElementsMatch(t, []string{"red", "yellow", "blue"}, tagMap.All("color"))
	require.Empty(t, tagMap.All("nonexistent"))
}
