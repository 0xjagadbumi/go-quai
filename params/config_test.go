// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"math/big"
	"reflect"
	"testing"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new *ChainConfig
		head        uint64
		wantErr     *ConfigCompatError
	}
	tests := []test{
		{stored: AllBlake3powProtocolChanges, new: AllBlake3powProtocolChanges, head: 0, wantErr: nil},
		{stored: AllBlake3powProtocolChanges, new: AllBlake3powProtocolChanges, head: 100, wantErr: nil},
		{
			stored:  &ChainConfig{EIP150Block: big.NewInt(10)},
			new:     &ChainConfig{EIP150Block: big.NewInt(20)},
			head:    9,
			wantErr: nil,
		},
		{
			stored:  &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:     &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(30)},
			head:    40,
			wantErr: nil,
		},
		{
			stored: &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:    &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(31)},
			head:   40,
			wantErr: &ConfigCompatError{
				What:         "Petersburg fork block",
				StoredConfig: nil,
				NewConfig:    big.NewInt(31),
				RewindTo:     30,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.head)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nhead: %v\nerr: %v\nwant: %v", test.stored, test.new, test.head, err, test.wantErr)
		}
	}
}
