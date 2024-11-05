package v8go

// #include <stdlib.h>
// #include "v8go.h"
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

// ExternalStore is a store for external values.
type ExternalStore struct {
	counter uintptr
	store   sync.Map
	lock    sync.Mutex
}

var externals = NewExternalStore()

// NewExternalStore creates a new ExternalStore.
func NewExternalStore() *ExternalStore {
	return &ExternalStore{
		counter: 0,
		store:   sync.Map{},
		lock:    sync.Mutex{},
	}
}

// Get returns a value from the store.
func (s *ExternalStore) Get(key uintptr) (interface{}, bool) {
	return s.store.Load(key)
}

// Add adds a value to the store. It returns a key that can be used to retrieve the value.
func (s *ExternalStore) Add(value interface{}) uintptr {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.counter++
	s.store.Store(s.counter, value)
	return s.counter
}

// Remove removes a value from the store.
func (s *ExternalStore) Remove(key uintptr) {
	s.store.Delete(key)
}

// ExternalCount returns the number of external values in the store.
func ExternalCount() int {
	count := 0
	externals.store.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// NewExternal creates a new external value.
func NewExternal(iso *Isolate, val interface{}) (*Value, error) {
	ptr := externals.Add(val)
	rtnVal := &Value{
		ptr: C.NewValueExternal(iso.ptr, unsafe.Pointer(ptr)),
	}
	return rtnVal, nil
}

// External returns the external value.
// then an error is returned. Use `value.Object()` to do the JS equivalent of `Object(value)`.
func (v *Value) External() (interface{}, error) {
	if !v.IsYaoExternal() {
		return nil, errors.New("v8go: value is not an External")
	}

	rtnValue := C.ValueToExternal(v.ptr)
	if rtnValue == 0 {
		return nil, errors.New("v8go: failed to get external value")
	}

	value, ok := externals.Get(uintptr(rtnValue))
	if !ok {
		return nil, errors.New("v8go: failed to get external value, not found")
	}

	return value, nil
}

// IsYaoExternal returns true if the value is an external value.
func (v *Value) IsYaoExternal() bool {
	return C.ValueIsExternal(v.ptr) != 0
}

// ReleaseExternal releases the external value.
func (v *Value) ReleaseExternal() {
	if v.IsYaoExternal() {
		rtnValue := C.ValueToExternal(v.ptr)
		if rtnValue != 0 {
			externals.Remove(uintptr(rtnValue))
		}
	}
}
