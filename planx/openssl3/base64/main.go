package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	fi, err := os.Open(
		"../client/client.pfx")
	if err != nil {
		panic(fmt.Sprintf("JPInit Open %s", err.Error()))
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	if err != nil {
		panic(fmt.Sprintf("JPInit ReadAll %s", err.Error()))
	}

	en := base64.NewEncoding(CharacterSetBase)

	encoded := en.EncodeToString(fd)
	fmt.Println(encoded)
}

const CharacterSetBase = "ja-1qD40zXBO5kNHfcUiTYgVlMeQwLP7bmEsxo3RW_ZKFSnA86hGtIy9rdpC2uvJ"
