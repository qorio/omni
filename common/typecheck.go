package common

import (
	"reflect"
)

// A simple check of type comparison.  Ptr to a struct and struct are
// considered compatible.  This is useful mostly for the cases where
// we have interface implementations
func TypeMatch(this interface{}, that interface{}) bool {
	t1 := reflect.TypeOf(this)
	t2 := reflect.TypeOf(that)
	if t1 == t2 {
		return true
	}
	if reflect.PtrTo(t1) == t2 || t1 == reflect.PtrTo(t2) {
		return true
	}
	return false
}
