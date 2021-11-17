package relayer

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type CreateProfileResponse struct {
	Success             bool           `json:"success"`
	TaskID              string         `json:"taskId"`
	Erc725ControllerKey common.Address `json:"erc725ControllerKey"`
	Salt                string         `json:"salt"`
	ProfileLinkHex      string         `json:"profileLinkHex"`
	Err                 string         `json:"error,omitempty"`
}

type CreateProfileTaskResponse struct {
	Success bool                `json:"success"`
	TaskID  string              `json:"taskId"`
	Status  string              `json:"status"`
	Err     string              `json:"error"`
	Data    ProfileDataResponse `json:"data"`
}

type ProfileDataResponse struct {
	Contracts ProfileDataContractsResponse `json:"contracts"`
	Message   interface{}                  `json:"message"`
}

type ProfileDataContractsResponse struct {
	Erc725            GenericContractTaskResponse `json:"erc725"`
	KeyManager        GenericContractTaskResponse `json:"keyManager"`
	UniversalReceiver GenericContractTaskResponse `json:"universalReceiver"`
}

type GenericContractTaskResponse struct {
	Block           big.Int                 `json:"block"`
	Status          string                  `json:"string"`
	Address         common.MixedcaseAddress `json:"address"`
	Version         string                  `json:"version"`
	TransactionHash common.Hash             `json:"transactionHash"`
}
