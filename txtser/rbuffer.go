// Copyright 2018 The go-boostio Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txtser

import (
	"io"
	"reflect"
	"strconv"
)

// A RBuffer reads values from a Boost textual serialization stream.
type RBuffer struct {
	r    io.Reader
	err  error
	buf  []byte
	arch Arch

	types registry
}

// NewRBuffer returns a new read-only buffer that reads from r.
func NewRBuffer(r io.Reader) *RBuffer {
	return &RBuffer{
		r:     r,
		buf:   make([]byte, 8),
		types: newRegistry(),
	}
}

func (r *RBuffer) Err() error { return r.err }

func (r *RBuffer) ReadHeader() Header {
	var hdr Header
	if r.r == nil {
		r.err = ErrNotBoost
		return hdr
	}

	if r.err != nil {
		return hdr
	}

	// "22 serialization::archive". where "22" is the length of the magic signature
	v := r.ReadString()
	if r.err != nil {
		r.err = ErrNotBoost
		return hdr
	}

	if v != magicHeaderSignature {
		r.err = ErrNotBoost
		return hdr
	}

	/*
		var (
			sz = len(magicHeaderSignature)
			// v  = string(r.buf[4:])
		)
		switch {
		case v == magicHeaderSignature[:4]:
			sz -= 4
			r.arch = Arch32
		default:
			v = ""
			r.arch = Arch64
		}
		raw := make([]byte, sz)
		_, _ = r.Read(raw)
		if r.err != nil {
			r.err = ErrNotBoost
			return hdr
		}
		v += string(raw)
	*/
	hdr.UnmarshalBoost(r)
	if r.err != nil {
		r.err = ErrInvalidHeader
	}
	return hdr
}

func (r *RBuffer) ReadTypeDescr(typ reflect.Type) TypeDescr {
	if dtype, ok := r.types[typ]; ok {
		return dtype
	}

	var dtype TypeDescr
	dtype.UnmarshalBoost(r)
	switch r.err {
	case nil:
		r.types[typ] = dtype
	default:
		r.err = ErrInvalidTypeDescr
	}
	return dtype
}

func (r *RBuffer) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	var n int
	n, r.err = io.ReadFull(r.r, p)
	return n, r.err
}

func (r *RBuffer) readLen() int {
	switch r.arch {
	case 32:
		return int(r.ReadU32())
	default:
		return int(r.ReadU64())
	}
}

func (r *RBuffer) skipDelimiter() {
	b := []byte{0}
	n, _ := r.r.Read(b)
	if n > 0 {
		s := string(b)
		if s != " " {
			r.err = ErrNotADelimiter
		}
	}

}

func (r *RBuffer) ReadString() string {
	n := r.readLen()
	if n == 0 || r.err != nil {
		return ""
	}
	raw := make([]byte, n)
	_, r.err = io.ReadFull(r.r, raw)
	r.skipDelimiter() // skip the delimiter following a string - does *not* set EOF on end ...
	return string(raw)
}

func (r *RBuffer) ReadBool() bool {

	switch r.ReadU8() {
	case 0:
		return false
	default:
		return true
	}
}

func (r *RBuffer) ReadU8() uint8 {
	return uint8(r.ReadU64())
}

func (r *RBuffer) ReadU16() uint16 {
	return uint16(r.ReadU64())
}

func (r *RBuffer) ReadU32() uint32 {
	return uint32(r.ReadU64())
}

func (r *RBuffer) ReadU64() uint64 {
	if r.err != nil {
		return 0
	}
	s := r.readUntilDelimiter()
	if r.err != nil {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		r.err = err
		return 0
	}
	return v
}

func (r *RBuffer) ReadI8() int8 {
	return int8(r.ReadI64())
}

func (r *RBuffer) ReadI16() int16 {
	return int16(r.ReadI64())
}

func (r *RBuffer) ReadI32() int32 {
	return int32(r.ReadI64())
}

func (r *RBuffer) ReadI64() int64 {
	if r.err != nil {
		return 0
	}
	s := r.readUntilDelimiter()
	if r.err != nil {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		r.err = err
		return 0
	}
	return v
}

func (r *RBuffer) ReadF64() float64 {
	if r.err != nil {
		return 0
	}
	s := r.readUntilDelimiter()
	if r.err != nil {
		return 0
	}
	// v, err := strconv.ParseInt(s, 10, 64)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		r.err = err
		return 0
	}
	return v
}

func (r *RBuffer) ReadF32() float32 {
	return float32(r.ReadF64())
}

func (r *RBuffer) ReadC64() complex64 {
	v0 := r.ReadF32()
	v1 := r.ReadF32()
	return complex(v0, v1)
}

func (r *RBuffer) ReadC128() complex128 {
	v0 := r.ReadF64()
	v1 := r.ReadF64()
	return complex(v0, v1)
}

/*
func (r *RBuffer) load(n int) {
	if r.err != nil {
		return
	}

	nn, err := io.ReadFull(r.r, r.buf[:n])
	if err != nil {
		r.err = err
		return
	}

	if nn < n {
		r.err = io.ErrUnexpectedEOF
	}
}
*/

func (r *RBuffer) readUntilDelimiter() string {
	if r.err != nil {
		return ""
	}

	ret := ""
	var n int = 0
	var err error
	for {
		b := []byte{0}
		n, err = r.r.Read(b)
		if err != nil {
			break
		}
		n++
		s := string(b)
		if s == " " {
			break
		}
		ret += s
	}
	if n == 0 {
		r.err = io.ErrUnexpectedEOF
		return ""
	}

	return ret
}

var (
	_ io.Reader = (*RBuffer)(nil)
)
