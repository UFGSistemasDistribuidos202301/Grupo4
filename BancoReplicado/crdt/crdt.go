package crdt

import (
	"time"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Hybrid Logical Clock implemented as in: https://jaredforsyth.com/posts/hybrid-logical-clocks/
// Implements a Last-Writer-Wins semantic, with more priority given to the greater NodeID

type HLC struct {
	Timestamp int64
	Counter   int
	NodeID    int
}

func (local HLC) Increment() HLC {
	now := time.Now().UnixNano()
	if now > local.Timestamp {
		local.Timestamp = now
		local.Counter = 0
	} else {
		local.Counter++
	}
	return local
}

func (p1 HLC) Cmp(p2 HLC) int {
	if p1.Timestamp == p2.Timestamp {
		if p1.Counter == p2.Counter {
			if p1.NodeID > p2.NodeID {
				return 1
			} else if p1.NodeID < p2.NodeID {
				return -1
			} else {
				return 0
			}
		}
		return p1.Counter - p2.Counter
	}
	return int(p1.Timestamp - p2.Timestamp)
}

func (local HLC) Merge(remote HLC) HLC {
	now := time.Now().UnixNano()
	if now > local.Timestamp && now > remote.Timestamp {
		local.Timestamp = now
		local.Counter = 0
		return local
	} else if local.Timestamp == remote.Timestamp {
		local.Counter = maxInt(local.Counter, remote.Counter) + 1
		return local
	} else if local.Timestamp > remote.Timestamp {
		local.Counter++
		return local
	} else {
		local.Timestamp = remote.Timestamp
		local.Counter = remote.Counter + 1
		return local
	}
}

func NowHLC(nodeID int) HLC {
	return HLC{
		Timestamp: time.Now().UnixNano(),
		Counter:   0,
		NodeID:    nodeID,
	}
}

func NullHLC() HLC {
	return HLC{
		Timestamp: 0,
		Counter:   0,
		NodeID:    0,
	}
}

///////////////////////////////////////////////////////////////////////////////

type MergeableString struct {
	Value string
	Point HLC
}

func (m MergeableString) Merge(other MergeableString) MergeableString {
	newPoint := m.Point.Merge(other.Point)
	if m.Point.Cmp(other.Point) >= 0 {
		m.Point = newPoint
		return m
	}
	other.Point = newPoint
	return other
}

///////////////////////////////////////////////////////////////////////////////

type MergeableMap struct {
	Point      HLC
	Map        map[string]MergeableString
	Tombstones map[string]HLC
}

func (m MergeableMap) Put(key, value string) MergeableMap {
	m2 := m.clone()
	m2.Point = m2.Point.Increment()

	m2.Map[key] = MergeableString{
		Point: m2.Point,
		Value: value,
	}
	delete(m2.Tombstones, key)

	return m2
}

func (m MergeableMap) Remove(key string) MergeableMap {
	m2 := m.clone()
	m2.Point = m2.Point.Increment()

	m2.Tombstones[key] = m2.Point
	delete(m2.Map, key)

	return m2
}

func (m *MergeableMap) Get(key string) (string, bool) {
	val, ok := m.Map[key]
	return val.Value, ok
}

func (m *MergeableMap) Len() int {
	return len(m.Map)
}

func (m MergeableMap) Merge(other MergeableMap) MergeableMap {
	allTombstones := mergeTombstones(m.Tombstones, other.Tombstones)
	elements := map[string]MergeableString{}

	iter := func(key string) {
		left, leftOk := m.Map[key]
		right, rightOk := other.Map[key]

		if !leftOk && !rightOk {
			panic("This should never happen")
		}

		var winner MergeableString
		if leftOk && !rightOk {
			winner = left
		} else if !leftOk && rightOk {
			winner = right
		} else {
			winner = left.Merge(right)
		}

		if _, ok := allTombstones[key]; !ok {
			// There are no tombstones for this item, so we can add it
			elements[key] = winner
		} else if winner.Point.Cmp(allTombstones[key]) >= 0 {
			// The winner is newer than the tombstone, so we remove the tombstone and add the winner
			delete(allTombstones, key)
			elements[key] = winner
		}
	}

	for key := range m.Map {
		iter(key)
	}
	for key := range other.Map {
		iter(key)
	}

	return MergeableMap{
		Point:      m.Point.Merge(other.Point),
		Map:        elements,
		Tombstones: allTombstones,
	}
}

func mergeTombstones(ts1 map[string]HLC, ts2 map[string]HLC) map[string]HLC {
	iter := func(key string) HLC {
		p1 := NullHLC()
		if p, ok := ts1[key]; ok {
			p1 = p
		}
		p2 := NullHLC()
		if p, ok := ts2[key]; ok {
			p2 = p
		}
		return p1.Merge(p2)
	}

	mergedTombstones := map[string]HLC{}
	for key := range ts1 {
		latestPoint := iter(key)
		mergedTombstones[key] = latestPoint
	}
	for key := range ts2 {
		latestPoint := iter(key)
		mergedTombstones[key] = latestPoint
	}

	return mergedTombstones
}

func (m MergeableMap) clone() MergeableMap {
	m2 := MergeableMap{
		Point:      m.Point,
		Map:        map[string]MergeableString{},
		Tombstones: map[string]HLC{},
	}
	for k, v := range m.Map {
		m2.Map[k] = v
	}
	for k, v := range m.Tombstones {
		m2.Tombstones[k] = v
	}
	return m2
}

func NewMergeableMap(NodeID int) MergeableMap {
	return MergeableMap{
		Point: HLC{
			Timestamp: 0,
			Counter:   0,
			NodeID:    NodeID,
		},
		Map:        map[string]MergeableString{},
		Tombstones: map[string]HLC{},
	}
}
