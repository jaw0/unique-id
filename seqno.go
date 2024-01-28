// Copyright (c) 2024
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2024-Jan-28 12:24 (EST)
// Function: sequence generator

package id

import (
	"sync"
)

type sequence struct {
	lock  sync.Mutex
	mark  uint64
	seqno uint64
}

func (sq *sequence) Next(mark uint64) (uint64, []uint16) {

	sq.lock.Lock()
	defer sq.lock.Unlock()

	if mark < sq.mark {
		// use latest mark
		mark = sq.mark
	}

	if mark != sq.mark {
		// move ahead to new mark
		sq.seqno = 0
		sq.mark = mark
		return mark, []uint16{uint16(sq.seqno)}
	}

	// advance
	sq.seqno++

	if sq.seqno <= 0xffff {
		return mark, []uint16{uint16(sq.seqno)}
	}

	// need longer seq
	seq := sq.seqno
	seqs := []uint16{}

	for seq != 0 {
		s := uint16(seq)
		if len(seqs) > 0 {
			s = s ^ uint16(mark)
		}
		seqs = append(seqs, s)
		seq = seq >> 16
	}

	return mark, seqs
}
