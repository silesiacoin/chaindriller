package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/silesiacoin/chaindriller/relayer"
	"log"
)

func main() {
	log.Println("deploy of the UP")
	client := relayer.NewStaging()
	addr, err := common.NewMixedcaseAddressFromString("0xE6D95f736b2e89B9b6062EF7c39ea740B4801D85")

	if nil != err {
		log.Fatalln(err)
		return
	}

	log.Println("started new client")
	err, resp := client.CreateProfile(
		relayer.DefaultProfileJson,
		addr,
		"dummys@lukso.io",
	)

	log.Println("after response")

	if nil != err {
		log.Fatalln(err)
		return
	}

	log.Println(resp)
}
