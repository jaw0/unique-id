// Copyright (c) 2018
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2018-Jul-09 11:57 (EST)
// Function:

package id

import (
	"fmt"
	"testing"
	"time"
)

func TestSeqno(t *testing.T) {
	sq := &sequence{}

	for i := 0; i < 1000000; i++ {
		m, s := sq.Next(1)

		if m != 1 {
			t.Errorf("expected a 1 mark, got %d", m)
		}
		exp := i >> 16

		if exp == 0 && len(s) != 1 {
			t.Errorf("expected len 1 seq: %v", s)
		}
		if exp != 0 && len(s) != 2 {
			t.Errorf("expected len 2 seq: %v", s)
		}
	}

	for i := 0; i < 1000000; i++ {
		m, s := sq.Next(2)

		if m != 2 {
			t.Errorf("expected a 2 mark, got %d", m)
		}
		exp := i >> 16

		if exp == 0 && len(s) != 1 {
			t.Errorf("expected len 1 seq: %v", s)
		}
		if exp != 0 && len(s) != 2 {
			t.Errorf("expected len 2 seq: %v", s)
		}
	}

}

func TestUnique(t *testing.T) {

	if Unique() == Unique() {
		t.Fail()
	}
	if len(Unique(WithLength(30))) != 30 {
		t.Fail()
	}
	if len(Unique(WithLength(41))) != 41 {
		t.Fail()
	}
	if len(Unique(WithUpperCase(), WithLength(30))) != 30 {
		t.Fail()
	}
	if len(Unique(WithUpperCase(), WithLength(41))) != 41 {
		t.Fail()
	}

	fmt.Printf("u %v\n", Unique())
	fmt.Printf("u %v\n", Unique())
	fmt.Printf("u %v\n", Unique())
	fmt.Printf("u %v\n", Unique())
	fmt.Printf("u %v\n", Unique(WithUpperCase()))
	fmt.Printf("u %v\n", Unique(WithUpperCase()))
	fmt.Printf("u %v\n", Unique(WithLength(30)))
	fmt.Printf("u %v\n", Unique(WithLength(30)))

	fmt.Printf("u %v\n", Unique(WithHost16(1)))
	fmt.Printf("u %v\n", Unique(WithHost16(1)))

	scramble(0, 0, 0, 1)
	scramble(1, 2, 3, 4)

	if Unique() == Unique() {
		t.Fail()
	}
}

func TestUniques(t *testing.T) {
	checkDupes(t, func() string { return Unique() })

	g := NewGenerator(WithUpperCase())
	checkDupes(t, func() string { return g.Unique() })

	g = NewGenerator(WithHost16(1))
	checkDupes(t, func() string { return g.Unique() })

	g = NewGenerator(WithLength(1))
	checkDupes(t, func() string { return g.Unique() })
}

func checkDupes(t *testing.T, fn func() string) {
	seen := make(map[string]time.Time)

	for i := 0; i < 1000000; i++ {
		u := fn()
		if x, ok := seen[u]; ok {
			t.Errorf("dupe! %s [%+v]", u, x)
		}
		seen[u] = time.Now().UTC()
	}
}
