package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/silesiacoin/chaindriller/relayer"
	"log"
)

func main() {
	log.Println("deploy of the UP")
	client := relayer.NewStaging()
	log.Println("started new client")
	resp, err := client.CreateProfile(
		relayer.DefaultProfileJson,
		common.HexToAddress("0xd3d39CB1d8aDA2bF1260FE05B7358DC031812b3a"),
		"dummy@lukso.io",
	)

	log.Println("after response")

	if nil != err {
		log.Fatalln(err)
	}

	log.Println(resp)
}
