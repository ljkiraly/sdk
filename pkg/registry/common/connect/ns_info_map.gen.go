// Code generated by "-output ns_info_map.gen.go -type nsInfoMap<string,*nsInfo> -output ns_info_map.gen.go -type nsInfoMap<string,*nsInfo>"; DO NOT EDIT.
package connect

import (
	"sync" // Used by sync.Map.
)

// Generate code that will fail if the constants change value.
func _() {
	// An "cannot convert nsInfoMap literal (type nsInfoMap) to type sync.Map" compiler error signifies that the base type have changed.
	// Re-run the go-syncmap command to generate them again.
	_ = (sync.Map)(nsInfoMap{})
}

var _nil_nsInfoMap_nsInfo_value = func() (val *nsInfo) { return }()

// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (m *nsInfoMap) Load(key string) (*nsInfo, bool) {
	value, ok := (*sync.Map)(m).Load(key)
	if value == nil {
		return _nil_nsInfoMap_nsInfo_value, ok
	}
	return value.(*nsInfo), ok
}

// Store sets the value for a key.
func (m *nsInfoMap) Store(key string, value *nsInfo) {
	(*sync.Map)(m).Store(key, value)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *nsInfoMap) LoadOrStore(key string, value *nsInfo) (*nsInfo, bool) {
	actual, loaded := (*sync.Map)(m).LoadOrStore(key, value)
	if actual == nil {
		return _nil_nsInfoMap_nsInfo_value, loaded
	}
	return actual.(*nsInfo), loaded
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *nsInfoMap) LoadAndDelete(key string) (value *nsInfo, loaded bool) {
	actual, loaded := (*sync.Map)(m).LoadAndDelete(key)
	if actual == nil {
		return _nil_nsInfoMap_nsInfo_value, loaded
	}
	return actual.(*nsInfo), loaded
}

// Delete deletes the value for a key.
func (m *nsInfoMap) Delete(key string) {
	(*sync.Map)(m).Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *nsInfoMap) Range(f func(key string, value *nsInfo) bool) {
	(*sync.Map)(m).Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*nsInfo))
	})
}