// Copyright 2018 The go-boostio Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txtser

import (
	"io"
	"reflect"
	"sync"
)

// An Encoder writes and encodes values to a Boost textual serialization stream.
type Encoder struct {
	w      *WBuffer
	Header Header

	hdr sync.Once
}

// NewEncoder returns a new encoder that writes to w.
//
// The encoder writes a correct Boost textual header at the beginning of
// the archive.
func NewEncoder(w io.Writer) *Encoder {
	return newEncoder(w, Arch64)
}

func newEncoder(w io.Writer, arch Arch) *Encoder {
	ww := newWBuffer(w, arch)
	return &Encoder{w: ww}
}

func (enc *Encoder) writeHeader() {
	if enc.Header == zeroHdr {
		enc.Header = enc.w.arch.Header()
	}

	enc.w.WriteString(magicHeaderSignature)
	enc.w.WriteHeader(enc.Header)
}

// Encode write the value v to its output.
func (enc *Encoder) Encode(v interface{}) error {
	enc.hdr.Do(enc.writeHeader)
	if enc.w.err != nil {
		return enc.w.err
	}

	if v, ok := v.(Marshaler); ok {
		return v.MarshalBoost(enc.w)
	}

	rv := reflect.Indirect(reflect.ValueOf(v))
	switch rv.Kind() {
	case reflect.Bool:
		enc.w.WriteBool(rv.Bool())
	case reflect.Int8:
		enc.w.WriteI8(int8(rv.Int()))
	case reflect.Int16:
		enc.w.WriteI16(int16(rv.Int()))
	case reflect.Int32:
		enc.w.WriteI32(int32(rv.Int()))
	case reflect.Int64:
		enc.w.WriteI64(rv.Int())
	case reflect.Uint8:
		enc.w.WriteU8(uint8(rv.Uint()))
	case reflect.Uint16:
		enc.w.WriteU16(uint16(rv.Uint()))
	case reflect.Uint32:
		enc.w.WriteU32(uint32(rv.Uint()))
	case reflect.Uint64:
		enc.w.WriteU64(rv.Uint())
	case reflect.Float32:
		enc.w.WriteF32(float32(rv.Float()))
	case reflect.Float64:
		enc.w.WriteF64(rv.Float())
	case reflect.Complex64:
		enc.w.WriteC64(complex64(rv.Complex()))
	case reflect.Complex128:
		enc.w.WriteC128(rv.Complex())
	case reflect.String:
		enc.w.WriteString(rv.String())
	case reflect.Struct:
		rt := rv.Type()
		enc.w.WriteTypeDescr(rt)
		for i := 0; i < rt.NumField(); i++ {
			enc.Encode(rv.Field(i).Interface())
		}
	case reflect.Slice:
		rt := rv.Type()
		enc.w.WriteTypeDescr(rt)
		n := rv.Len()
		enc.w.writeLen(n)
		if et := rt.Elem(); !isCxxBoostBuiltin(et.Kind()) {
			enc.w.WriteU32(0) // FIXME(sbinet): what is this?
		}
		for i := 0; i < int(n); i++ {
			e := rv.Index(i)
			enc.Encode(e.Interface()) // FIXME(sbinet): do not go through Decode each time
		}
	case reflect.Array:
		rt := rv.Type()
		enc.w.WriteTypeDescr(rt)
		n := int(rv.Len())
		enc.w.writeLen(n)
		for i := 0; i < n; i++ {
			e := rv.Index(i)
			enc.Encode(e.Interface()) // FIXME(sbinet): do not go through Decode each time
		}
	case reflect.Map:
		rt := rv.Type()
		enc.w.WriteTypeDescr(rt)
		n := int(rv.Len())
		enc.w.writeLen(n)
		enc.w.WriteU64(0) // FIXME(sbinet): what is this ?
		enc.w.WriteU8(0)  // FIXME(sbinet): ditto ?
		keys := rv.MapKeys()
		for _, k := range keys {
			v := rv.MapIndex(k)
			enc.Encode(k.Interface()) // FIXME(sbinet): do not go through Decode each time
			enc.Encode(v.Interface()) // FIXME(sbinet): do not go through Decode each time
		}

	default:
		return ErrTypeNotSupported
	}
	return enc.w.err
}
