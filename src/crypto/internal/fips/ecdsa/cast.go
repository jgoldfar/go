// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ecdsa

import (
	"bytes"
	"crypto/internal/fips"
	_ "crypto/internal/fips/check"
	"crypto/internal/fips/sha512"
	"errors"
	"sync"
)

func testPrivateKey() *PrivateKey {
	// https://www.rfc-editor.org/rfc/rfc9500.html#section-2.3
	return &PrivateKey{
		pub: PublicKey{
			curve: p256,
			q: []byte{
				0x04,
				0x42, 0x25, 0x48, 0xF8, 0x8F, 0xB7, 0x82, 0xFF,
				0xB5, 0xEC, 0xA3, 0x74, 0x44, 0x52, 0xC7, 0x2A,
				0x1E, 0x55, 0x8F, 0xBD, 0x6F, 0x73, 0xBE, 0x5E,
				0x48, 0xE9, 0x32, 0x32, 0xCC, 0x45, 0xC5, 0xB1,
				0x6C, 0x4C, 0xD1, 0x0C, 0x4C, 0xB8, 0xD5, 0xB8,
				0xA1, 0x71, 0x39, 0xE9, 0x48, 0x82, 0xC8, 0x99,
				0x25, 0x72, 0x99, 0x34, 0x25, 0xF4, 0x14, 0x19,
				0xAB, 0x7E, 0x90, 0xA4, 0x2A, 0x49, 0x42, 0x72},
		},
		d: []byte{
			0xE6, 0xCB, 0x5B, 0xDD, 0x80, 0xAA, 0x45, 0xAE,
			0x9C, 0x95, 0xE8, 0xC1, 0x54, 0x76, 0x67, 0x9F,
			0xFE, 0xC9, 0x53, 0xC1, 0x68, 0x51, 0xE7, 0x11,
			0xE7, 0x43, 0x93, 0x95, 0x89, 0xC6, 0x4F, 0xC1,
		},
	}
}

func testHash() []byte {
	return []byte{
		0x17, 0x1b, 0x1f, 0x5e, 0x9f, 0x8f, 0x8c, 0x5c,
		0x42, 0xe8, 0x06, 0x59, 0x7b, 0x54, 0xc7, 0xb4,
		0x49, 0x05, 0xa1, 0xdb, 0x3a, 0x3c, 0x31, 0xd3,
		0xb7, 0x56, 0x45, 0x8c, 0xc2, 0xd6, 0x88, 0x62,
		0x9e, 0xd6, 0x7b, 0x9b, 0x25, 0x68, 0xd6, 0xc6,
		0x18, 0x94, 0x1e, 0xfe, 0xe3, 0x33, 0x78, 0xa6,
		0xe1, 0xce, 0x13, 0x88, 0x81, 0x26, 0x02, 0x52,
		0xdf, 0xc2, 0x0a, 0xf2, 0x67, 0x49, 0x0a, 0x20,
	}
}

func fipsPCT[P Point[P]](c *Curve[P], k *PrivateKey) error {
	return fips.PCT("ECDSA PCT", func() error {
		hash := testHash()
		sig, err := Sign(c, sha512.New, k, nil, hash)
		if err != nil {
			return err
		}
		return Verify(c, &k.pub, hash, sig)
	})
}

var fipsSelfTest = sync.OnceFunc(func() {
	fips.CAST("ECDSA P-256 SHA2-512 sign and verify", func() error {
		k := testPrivateKey()
		Z := []byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		}
		persStr := []byte{
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
			0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		}
		hash := testHash()
		want := &Signature{
			R: []byte{
				0x33, 0x64, 0x96, 0xff, 0x8a, 0xfe, 0xaa, 0x0b,
				0x2c, 0x4a, 0x1a, 0x97, 0x77, 0xcc, 0x84, 0xa5,
				0x7e, 0x88, 0x1f, 0x16, 0x2d, 0xe0, 0x29, 0xf7,
				0x62, 0xc2, 0x34, 0x18, 0x10, 0x9c, 0x69, 0x8a,
			}, S: []byte{
				0x97, 0x53, 0x2e, 0x13, 0x6e, 0xd0, 0x9b, 0x30,
				0x8a, 0xdf, 0x4f, 0xe0, 0x54, 0x82, 0x14, 0x83,
				0x5e, 0x93, 0xc7, 0x79, 0x4b, 0x18, 0xa3, 0xf1,
				0x8a, 0x60, 0xae, 0x52, 0x31, 0xe4, 0x2e, 0x4e,
			},
		}
		drbg := newDRBG(sha512.New, Z, nil, plainPersonalizationString(persStr))
		got, err := sign(P256(), k, drbg, hash)
		if err != nil {
			return err
		}
		if err := verify(P256(), &k.pub, hash, got); err != nil {
			return err
		}
		if !bytes.Equal(got.R, want.R) || !bytes.Equal(got.S, want.S) {
			return errors.New("unexpected result")
		}
		return nil
	})
})

var fipsSelfTestDeterministic = sync.OnceFunc(func() {
	fips.CAST("DetECDSA P-256 SHA2-512 sign", func() error {
		k := testPrivateKey()
		hash := testHash()
		want := &Signature{
			R: []byte{
				0x9f, 0xc3, 0x83, 0x32, 0x6e, 0xd9, 0x4f, 0x8e,
				0x24, 0xa0, 0x19, 0xef, 0x1d, 0x3a, 0xc3, 0x55,
				0xdd, 0x4b, 0x98, 0xae, 0x78, 0xa7, 0xaf, 0xd3,
				0xfd, 0xf3, 0x22, 0x1c, 0x8b, 0xd6, 0x11, 0x7b,
			}, S: []byte{
				0xd6, 0x52, 0x87, 0x41, 0x71, 0xbd, 0x66, 0xd1,
				0xaf, 0x6c, 0x61, 0xdd, 0xd8, 0xa7, 0xbb, 0xd2,
				0xf7, 0xd5, 0x47, 0x70, 0xe9, 0xe4, 0xac, 0x0a,
				0xb9, 0xfa, 0x0f, 0xbd, 0x3b, 0x9b, 0xc2, 0xfe,
			},
		}
		drbg := newDRBG(sha512.New, k.d, bits2octets(P256(), hash), nil)
		got, err := sign(P256(), k, drbg, hash)
		if err != nil {
			return err
		}
		if err := verify(P256(), &k.pub, hash, got); err != nil {
			return err
		}
		if !bytes.Equal(got.R, want.R) || !bytes.Equal(got.S, want.S) {
			return errors.New("unexpected result")
		}
		return nil
	})
})