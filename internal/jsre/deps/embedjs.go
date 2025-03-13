// Copyright 2025 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

// deps package embeds the js files needed for opt/console package jsre
package deps

import "embed"

//go:embed *.js
var embedded embed.FS

func MustAsset(name string) []byte {
	data, err := embedded.ReadFile(name)
	if err != nil {
		panic("embed.FS lookup " + name + ": " + err.Error())
	}
	return data
}
