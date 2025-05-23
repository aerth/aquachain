// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

package aquaflags

import (
	"os"
	"os/user"
	"testing"

	"gitlab.com/aquachain/aquachain/common/sense"
)

func TestPathExpansion(t *testing.T) {
	homedir := sense.Getenv("HOME")
	user, err := user.Current()
	if err != nil {
		t.Logf("could not get 'user', assuming $HOME=%q, error=%q", homedir, err.Error())
	} else {
		homedir = user.HomeDir
	}
	tests := map[string]string{
		"/home/someuser/tmp": "/home/someuser/tmp",
		"~/tmp":              homedir + "/tmp",
		"~thisOtherUser/b/":  "~thisOtherUser/b",
		"$DDDXXX/a/b":        "/tmp/a/b",
		"/a/b/":              "/a/b",
	}
	os.Setenv("DDDXXX", "/tmp")
	for test, expected := range tests {
		got := expandPath(test)
		if got != expected {
			t.Errorf("test %s, got %s, expected %s\n", test, got, expected)
		}
	}
}
