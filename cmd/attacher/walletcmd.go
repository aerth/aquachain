package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/crypto"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	paperCommand = cli.Command{
		Name:      "paper",
		Usage:     `Generate paper wallet keypair`,
		Flags:     []cli.Flag{utils.JsonFlag, utils.VanityFlag},
		ArgsUsage: "[number of wallets]",
		Category:  "ACCOUNT COMMANDS",
		Action:    paper,
		Description: `
Generate a number of wallets.`,
	}
)

type paperWallet struct{ Private, Public string }

func paper(c *cli.Context) error {

	if c.NArg() > 1 {
		return fmt.Errorf("too many arguments")
	}
	var (
		count = 1
		err   error
	)
	if c.NArg() == 1 {
		count, err = strconv.Atoi(c.Args().First())
		if err != nil {
			return err
		}
	}
	wallets := []paperWallet{}
	vanity := c.String("vanity")
	for i := 0; i < count; i++ {
		var wallet paperWallet
		for {
			key, err := crypto.GenerateKey()
			if err != nil {
				return err
			}

			addr := crypto.PubkeyToAddress(key.PublicKey)
			wallet = paperWallet{
				Private: hex.EncodeToString(crypto.FromECDSA(key)),
				Public:  "0x" + hex.EncodeToString(addr[:]),
			}
			if vanity == "" {
				break
			}
			pubkey := hex.EncodeToString(addr[:])
			if strings.HasPrefix(pubkey, vanity) {
				break
			}
		}
		if c.Bool("json") {
			wallets = append(wallets, wallet)
		} else {
			fmt.Println(wallet.Private, wallet.Public)
		}
	}
	if c.Bool("json") {
		b, _ := json.Marshal(wallets)
		fmt.Println(string(b))
	}
	return nil
}
