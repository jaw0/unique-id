// Copyright (c) 2018
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2018-Jul-07 11:20 (EST)
// Function: ac style unique id

// id - AC style unique ids
package id

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"net"
	"os"
	"time"
)

// Gen is a uniqueid generator
type Gen struct {
	upper    bool
	numChars int
	numBytes int
	addr32   uint32
	addr16   uint16
	pid      uint16
	seq      *sequence
}

var myaddr = myAddr()
var mypid = os.Getpid()
var seqgen = &sequence{}

// no {l,o,0,1}
var idEncoder = base32.NewEncoding("abcdefghijkmnpqrstuvwxyz23456789").WithPadding(base32.NoPadding)

type optFunc func(*Gen)

// NewGenerator returns a new uniqueid generator
func NewGenerator(opts ...optFunc) *Gen {
	cf := &Gen{
		addr32: myaddr,
		pid:    uint16(os.Getpid()),
		seq:    &sequence{},
	}
	for _, fnc := range opts {
		fnc(cf)
	}

	cf.calcBytes()

	return cf
}

// Unique() returns a unique id
// ids are url safe, non-sortable, and appear "random-looking" with no visible pattern
// eg: rk5zzm2zkmt5djecxwj4
// if passed the option WithUpperCase(), the id will be shorter and contain upper + lower case letters
// eg: KaGFYkoJhPs3eNx9
// if passed WithLength(n), the id will be no shorter than the requested length
func Unique(opts ...optFunc) string {
	cf := &Gen{
		addr32: myaddr,
		pid:    uint16(mypid),
		seq:    seqgen,
	}
	for _, fnc := range opts {
		fnc(cf)
	}

	cf.calcBytes()
	return cf.Unique()
}

// Unique() returns a unique id
func (cf *Gen) Unique() string {

	uid, el := cf.unique()

	if cf.upper {
		id := base64.RawURLEncoding.EncodeToString(uid)
		return id[:cf.numChars+3*el]
	}

	id := idEncoder.EncodeToString(uid)
	return id[:cf.numChars+4*el]
}

func (cf *Gen) unique() ([]byte, int) {
	buf := &bytes.Buffer{}

	now := time.Now()
	var f1, f2 uint32
	var mark uint64

	if cf.addr16 != 0 {
		// ~ 1/16s of a millisec
		mark = uint64(now.UnixNano() >> 16)
	} else {
		mark = uint64(now.Unix())
	}

	mark, seqs := cf.seq.Next(mark)

	if cf.addr16 != 0 {
		f1 = uint32(mark >> 16)
		f2 = uint32(cf.addr16) | uint32(mark<<16)
	} else {
		f2 = cf.addr32
		f1 = uint32(mark)
	}

	// id = encoded( scrambled( time + address + pid + counter ) )
	a, b, c, d := scramble(f1, f2, cf.pid, seqs[0])

	binary.Write(buf, binary.LittleEndian, a)
	binary.Write(buf, binary.LittleEndian, b)
	binary.Write(buf, binary.LittleEndian, c)
	binary.Write(buf, binary.LittleEndian, d)

	// need to extend?
	binary.Write(buf, binary.LittleEndian, seqs[1:])

	if buf.Len() < cf.numBytes {
		// extend with random data
		rbuf := make([]byte, cf.numBytes-buf.Len())
		rand.Read(rbuf)
		buf.Write(rbuf)
	} else {
		// 96 bits base32 encoded leaves 4 bits of zero padding
		// add an extra byte + remove it, to randomize the padding
		// otherwise the id will always end in 'q' or 'a'
		binary.Write(buf, binary.LittleEndian, uint8(d))
	}

	return buf.Bytes(), len(seqs) - 1
}

func (cf *Gen) calcBytes() {
	if cf.upper {
		cf.numBytes = (cf.numChars*6 + 7) >> 3
		if cf.numChars < 16 {
			cf.numChars = 16
		}
	} else {
		cf.numBytes = (cf.numChars*5 + 7) >> 3
		if cf.numChars < 20 {
			cf.numChars = 20
		}
	}
}

func myAddr() uint32 {

	addrs, _ := net.InterfaceAddrs()
	var ip4, ip16 net.IP

	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ip == nil {
			continue
		}

		if ip.IsLoopback() {
			continue
		}

		v4 := ip.To4()

		if len(v4) == 4 {
			if v4.IsGlobalUnicast() || !ip4.IsGlobalUnicast() {
				ip4 = v4
			}
		} else {
			if ip.IsGlobalUnicast() || !ip16.IsGlobalUnicast() {
				ip16 = ip
			}
		}
	}

	if ip4 != nil {
		return packAddr(ip4)
	}

	if ip16 != nil {
		return packAddr(ip16[12:16])
	}

	// use a random value
	r := make([]byte, 4)
	rand.Read(r)
	return packAddr(r)
}

func packAddr(a []byte) uint32 {

	var n uint32
	for _, v := range a {
		n = n<<8 | uint32(v)
	}

	return n
}

// simple 2-round 96-bit feistel
func scramble(a uint32, b uint32, c uint16, d uint16) (uint32, uint32, uint16, uint16) {

	h := uint64(a) | uint64(c)<<32
	l := uint64(b) | uint64(d)<<32

	h, l = scrambleStep(h, l)
	h, l = scrambleStep(h, l)

	x := uint16(h >> 32)
	y := uint16(l >> 32)

	//fmt.Printf("=> %x %x %x %x => %x %x %x %x\n", a, b, c, d, uint32(l), uint32(h), y, x)
	return uint32(l), uint32(h), y, x

}

const fortyeightbits = (1 << 48) - 1

func scrambleStep(h, l uint64) (uint64, uint64) {

	f := l | 0x100000420008

	f *= 0xcc9e2d51
	f += f >> 48
	f += 0xe6546b64
	f = (f << 49) | (f >> 15)
	f *= (f >> 32) ^ f
	f += f >> 48

	f ^= (l >> 24) | (l << 24)
	f &= fortyeightbits

	return l, (f ^ h)
}

// WithUpperCase causes the unique id to use uppercase and some (url safe) punctuation
// in addition to the default of lowercase + numbers
func WithUpperCase() func(*Gen) {
	return func(cf *Gen) {
		cf.upper = true
	}
}

// WithLength causes the unique id to be at least the requested length
// by adding randomly generated data if needed
func WithLength(n int) func(*Gen) {
	return func(cf *Gen) {
		cf.numChars = n
	}
}

// WithHost specifies a value unique to this server
// by default the host's IP address is used
func WithHost(host uint32) func(*Gen) {
	return func(cf *Gen) {
		cf.addr32 = host
	}
}

// WithHost16 specifies a value unique to this server
// by using a 16bit value, additional bits are available for the time field
// and more ids can be generated without the length growing
func WithHost16(host uint16) func(*Gen) {
	return func(cf *Gen) {
		cf.addr16 = host
	}
}

// WithHost16Default uses the lower 16bits of host's IP
// see also: WithHost16()
func WithHost16Default() func(*Gen) {
	return func(cf *Gen) {
		cf.addr16 = uint16(myaddr)
	}
}

// WithPid specifies a value unique to this process
// by default the lower 16bits of the process-id are used
func WithPid(pid uint16) func(*Gen) {
	return func(cf *Gen) {
		cf.pid = pid
	}
}

// CheckAddr returns the default 32bit host id (IP address), for examination
func CheckAddr() uint32 {
	return myaddr
}
