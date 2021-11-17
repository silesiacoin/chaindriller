package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/silesiacoin/chaindriller/relayer"
	"log"
)

func main() {
	log.Println("deploy of the UP")
	client := relayer.NewStaging()
	addr, err := common.NewMixedcaseAddressFromString("0xD3d39cb1d8ADa2bF1260fE05b7358dC031812B3A")

	if nil != err {
		log.Fatalln(err)
		return
	}

	log.Println("started new client")
	err, resp := client.CreateProfile(
		relayer.DefaultProfileJson,
		addr,
		"dummy@lukso.io",
	)

	log.Println("after response")

	if nil != err {
		log.Fatalln(err)
		return
	}

	log.Println(resp)
}
