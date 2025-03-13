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
