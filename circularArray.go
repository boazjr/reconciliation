package main

import "iter"

type CircularArray[T any] struct {
	data     []T
	head     int
	tail     int
	size     int
	capacity int
}

func NewCircularArray[T any](capacity int) *CircularArray[T] {
	return &CircularArray[T]{
		data:     make([]T, capacity),
		head:     0,
		tail:     0,
		size:     0,
		capacity: capacity,
	}
}

/*
[1]
[1,2]
[3,2]
*/
func (c *CircularArray[T]) Add(item T) {
	incTail := c.head == c.tail && c.size > 0
	c.data[c.head] = item
	c.head = (c.head + 1) % c.capacity
	if incTail {
		c.tail = c.head
		return
	}
	c.size++
}

func (c *CircularArray[T]) AllItr() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range c.size {
			if !yield(c.data[(c.tail+i)%c.capacity]) {
				return
			}
		}
	}
}

func (c *CircularArray[T]) All() []T {
	itr := c.AllItr()
	ret := make([]T, 0, c.size)
	for v := range itr {
		ret = append(ret, v)
	}
	return ret
}

func (c *CircularArray[T]) Last(x int) []T {
	itr := c.LastItr(x)
	if c.size < x {
		x = c.size
	}
	ret := make([]T, 0, x)
	for v := range itr {
		ret = append(ret, v)
	}
	return ret
}

// LastItr returns the last x items or less if there aren't enough items
func (c *CircularArray[T]) LastItr(x int) iter.Seq[T] {
	if x > c.size {
		x = c.size
	}
	return func(yield func(T) bool) {
		for i := range x {
			if !yield(c.data[(c.head-x+i+c.capacity)%c.capacity]) {
				return
			}
		}
	}
}

func (c *CircularArray[T]) IsFull() bool {
	return c.size == c.capacity
}

func (c *CircularArray[T]) Size() int {
	return c.size
}
