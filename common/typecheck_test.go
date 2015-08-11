package common

import (
	"github.com/bmizerany/assert"
	"testing"
)

type test1 struct{}
type test2 struct{ foo string }
type inf1 interface {
	IsInf1() bool
}

func (this *test1) IsInf1() bool {
	return true
}

func TestTypeMatch(t *testing.T) {
	assert.Equal(t, true, TypeMatch(test1{}, test1{}))
	assert.Equal(t, true, TypeMatch(&test1{}, test1{}))
	assert.Equal(t, true, TypeMatch(&test1{}, &test1{}))
	assert.Equal(t, true, TypeMatch(test2{}, test2{}))
	assert.Equal(t, true, TypeMatch(&test2{}, test2{}))
	assert.Equal(t, false, TypeMatch(test1{}, test2{}))
	assert.Equal(t, false, TypeMatch(1, test2{}))
	assert.Equal(t, false, TypeMatch(1, test2{}))
}
