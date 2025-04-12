package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CicrularArray(t *testing.T) {

	t.Run("add 2", func(t *testing.T) {
		ci := NewCircularArray[int](4)

		ci.Add(1)
		for v := range ci.AllItr() {
			assert.Equal(t, v, 1)
		}

		ci.Add(1)
		for v := range ci.AllItr() {
			assert.Equal(t, v, 1)
		}
		assert.Equal(t, 2, ci.Size())

	})

	/*
		[4,1,2,3] return [1,2,3,4] heads at 1
		last 3 items
		(1 - 3+4)%4 = 2
		last 2 items
		(1 - 2+4)%4 = 3

		[1,2,3]
		last 3 items
		(3 - 3+6)%6 = 0

		doesn't work when we try to return more items than size.
	*/

	t.Run("override tail", func(t *testing.T) {
		//override tail
		ci := NewCircularArray[int](2)
		ci.Add(1)
		ci.Add(2)
		ci.Add(3)
		// [3,2] heads at 1
		assert.Equal(t, 2, ci.Size())
		out := []int{}
		for v := range ci.AllItr() {
			out = append(out, v)
		}
		assert.Equal(t, []int{2, 3}, out)
	})

	t.Run("override tail", func(t *testing.T) {
		//override tail
		ci := NewCircularArray[int](2)
		ci.Add(1)
		ci.Add(2)
		ci.Add(3)
		// [3,2] heads at 1
		assert.Equal(t, 2, ci.Size())
		out := []int{}
		for v := range ci.AllItr() {
			out = append(out, v)
		}
		assert.Equal(t, []int{2, 3}, out)
	})

	t.Run("return last 3 with override", func(t *testing.T) {
		//override tail
		ci := NewCircularArray[int](4)
		ci.Add(0)
		ci.Add(1)
		ci.Add(2)
		ci.Add(3)
		ci.Add(4)
		// [3,2] heads at 1
		assert.Equal(t, 4, ci.Size())
		out := []int{}
		for v := range ci.LastItr(3) {
			out = append(out, v)
		}
		assert.Equal(t, []int{2, 3, 4}, out)
	})

	t.Run("return last 4 but size is 3 ", func(t *testing.T) {
		ci := NewCircularArray[int](6)
		ci.Add(0)
		ci.Add(1)
		ci.Add(2)
		// [3,2] heads at 1
		assert.Equal(t, 3, ci.Size())
		out := []int{}
		for v := range ci.LastItr(5) {
			out = append(out, v)
		}
		assert.Equal(t, []int{0, 1, 2}, out)
	})

}
