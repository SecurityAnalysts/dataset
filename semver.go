//
// Package dataset includes the operations needed for processing collections of JSON documents and their attachments.
//
// Authors R. S. Doiel, <rsdoiel@library.caltech.edu> and Tom Morrel, <tmorrell@library.caltech.edu>
//
// Copyright (c) 2019, Caltech
// All rights not granted herein are expressly reserved by Caltech.
//
// Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
package dataset

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Err holds Semver's error messages
type Err struct {
	Msg string
}

func (err *Err) Error() string {
	return err.Msg
}

// Semver holds the information to generate a semver string
type Semver struct {
	// Major version number (required, must be an integer as string)
	Major string `json:"major"`
	// Minor version number (required, must be an integer as string)
	Minor string `json:"minor"`
	// Patch level (optional, must be an integer as string)
	Patch string `json:"patch,omitempty"`
	// Suffix string, (optional, any string)
	Suffix string `json:"suffix,omitempty"`
	// Timestamp (optional, a timestamp in form of YYYY-MM-DD HH:MM:SS)
}

func (v *Semver) String() string {
	if v.Patch == "" {
		return "v" + v.Major + "." + v.Minor
	}
	if v.Suffix == "" {
		return "v" + v.Major + "." + v.Minor + "." + v.Patch
	}
	return "v" + v.Major + "." + v.Minor + "." + v.Patch + v.Suffix
}

// ToJSON takes a version struct and returns JSON as byte slice
func (v *Semver) ToJSON() []byte {
	src, _ := json.Marshal(v)
	return src
}

// ParseSemver takes a byte slice and returns a version struct,
// and an error value.
func ParseSemver(src []byte) (*Semver, error) {
	var (
		i   int
		err error
	)
	v := new(Semver)
	if bytes.HasPrefix(src, []byte("v")) {
		src = bytes.TrimPrefix(src, []byte("v"))
	}
	parts := strings.Split(string(src), ".")
	if len(parts) > 0 {
		i, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, &Err{Msg: "Major value must be an integer"}
		}
		v.Major = strconv.Itoa(i)
	} else {
		return nil, &Err{Msg: "Invalid version, expecting semver string"}
	}
	if len(parts) > 1 {
		i, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, &Err{Msg: "Minor value must be an integer"}
		}
		v.Minor = strconv.Itoa(i)
	} else {
		return nil, &Err{Msg: "Invalid version, expecting semver string"}
	}
	if len(parts) > 2 {
		i, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, &Err{Msg: "Patch value must be an integer"}
		}
		v.Patch = strconv.Itoa(i)
	}
	if len(parts) > 3 {
		v.Suffix = parts[3]
	}
	return v, nil
}