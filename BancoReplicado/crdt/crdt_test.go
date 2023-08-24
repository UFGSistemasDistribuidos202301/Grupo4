package crdt

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func stringTwoWayMerge(str1, str2 MergeableString, callback func(merged MergeableString)) {
	callback(str1.Merge(str2))
	callback(str2.Merge(str1))
}

func mapTwoWayMerge(m1, m2 MergeableMap, callback func(merged MergeableMap)) {
	callback(m1.Merge(m2))
	callback(m2.Merge(m1))
}

func TestStringSameTimestamp(t *testing.T) {
	str1 := MergeableString{Value: "hello", Point: HLC{Counter: 1, NodeID: 1}}
	str2 := MergeableString{Value: "world", Point: HLC{Counter: 1, NodeID: 2}}
	stringTwoWayMerge(str1, str2, func(merged MergeableString) {
		if merged.Value != "world" {
			t.Errorf("Expected 'world', got '%s'", merged.Value)
		}
	})
}

func TestStringDifferentTimestamp(t *testing.T) {
	str1 := MergeableString{Value: "hello", Point: HLC{Counter: 1, NodeID: 1}}
	str2 := MergeableString{Value: "world", Point: HLC{Counter: 2, NodeID: 2}}

	stringTwoWayMerge(str1, str2, func(merged MergeableString) {
		if merged.Value != "world" {
			t.Errorf("Expected 'world', got '%s'", merged.Value)
		}
	})
}

func TestMapBasic(t *testing.T) {
	m1 := NewMergeableMap(1)
	m2 := NewMergeableMap(2)

	m1 = m1.Put("key1", "value1")
	m2 = m2.Put("key2", "value2")

	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if val, ok := merged.Get("key1"); val != "value1" || !ok {
			t.Errorf("Expected 'value1', got '%s'", val)
		}
		if val, ok := merged.Get("key2"); val != "value2" || !ok {
			t.Errorf("Expected 'value2', got '%s'", val)
		}
	})

	m2 = m2.Remove("key1")
	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if _, ok := merged.Get("key1"); ok {
			t.Error("Expected 'key1' to be deleted")
		}
		if val, ok := merged.Get("key2"); val != "value2" || !ok {
			t.Errorf("Expected 'value2', got '%s'", val)
		}
	})

	m1 = m1.Remove("key2")
	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if _, ok := merged.Get("key1"); ok {
			t.Error("Expected 'key1' to be deleted")
		}
		if _, ok := merged.Get("key2"); ok {
			t.Error("Expected 'key2' to be deleted")
		}
	})

	m1 = m1.Put("key1", "one")
	m2 = m2.Put("key1", "two")
	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if val, ok := merged.Get("key1"); val != "two" || !ok {
			t.Errorf("Expected 'two', got '%s'", val)
		}
	})

	m1 = m1.Put("key1", "one")
	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if val, ok := merged.Get("key1"); val != "one" || !ok {
			t.Errorf("Expected 'one', got '%s'", val)
		}
	})

	m2 = m2.Remove("key1")
	mapTwoWayMerge(m1, m2, func(merged MergeableMap) {
		if _, ok := merged.Get("key1"); ok {
			t.Error("Expected 'key1' to be deleted")
		}
	})
}

func TestDelete(t *testing.T) {
	m1 := NewMergeableMap(1)
	m2 := NewMergeableMap(2)

	m1 = m1.Put("key1", "value1")

	m2 = m2.Merge(m1)
	if _, ok := m2.Get("key1"); !ok {
		t.Error("Expected 'key1' to exist")
	}

	time.Sleep(20 * time.Millisecond)

	m2 = m2.Remove("key1")
	if _, ok := m2.Get("key1"); ok {
		t.Error("Expected 'key1' to be deleted")
	}

	time.Sleep(20 * time.Millisecond)

	m1 = m1.Put("key1", "value2")
	if _, ok := m1.Get("key1"); !ok {
		t.Error("Expected 'key1' to exist")
	}

	time.Sleep(20 * time.Millisecond)

	m2 = m2.Merge(m1)
	if _, ok := m2.Get("key1"); !ok {
		t.Error("Expected 'key1' to exist")
	}
}

func TestMapFuzz(t *testing.T) {
	const N = 1000
	added := make(map[string]bool)
	removed := make(map[string]bool)

	generate := func(nodeID int) MergeableMap {
		m := NewMergeableMap(nodeID)
		for i := 0; i < N; i++ {
			is := fmt.Sprint(i)
			if rand.Float32() < 0.7 {
				added[is] = true
				m = m.Put(is, is)
			}
		}
		return m
	}

	remove := func(m MergeableMap) MergeableMap {
		for i := 0; i < N; i++ {
			is := fmt.Sprint(i)
			if rand.Float32() < 0.1 {
				removed[is] = true
				m = m.Remove(is)
			}
		}
		return m
	}

	maps := []MergeableMap{generate(1), generate(2), generate(3)}
	for i := 0; i < len(maps); i++ {
		maps[i] = remove(maps[i])
	}

	for i := 0; i < len(maps); i++ {
		maps[i] = remove(maps[i])
	}

	merged := maps[0].Merge(maps[1]).Merge(maps[2])

	finalMap := make(map[string]bool)
	for k := range added {
		finalMap[k] = true
	}
	for k := range removed {
		delete(finalMap, k)
	}

	if len(finalMap) != merged.Len() {
		t.Errorf("Expected %d, got %d", len(finalMap), merged.Len())
	}
}
