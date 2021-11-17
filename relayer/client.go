package relayer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"time"
)

const (
	StagingEndpoint    = `https://relayer.staging.lukso.dev`
	ProdEndpoint       = `https://relayer.lukso.network`
	DefaultProfileJson = `ipfs://QmecrGejUQVXpW4zS948pNvcnQrJ1KiAoM6bdfrVcWZsn5`
)

type Client struct {
	endpoint string
}

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

func New(endpoint string) (client *Client) {
	client = &Client{endpoint: endpoint}
	return
}

func NewStaging() (client *Client) {
	client = New(StagingEndpoint)
	return
}

func (client *Client) CreateProfile(
	profileJsonUrl string,
	erc725ControllerKey *common.MixedcaseAddress,
	email string,
) (
	err error,
	response *CreateProfileTaskResponse,
) {
	err, createProfileResponse := client.CreateProfileAsync(profileJsonUrl, erc725ControllerKey, email)

	if nil != err {
		return
	}

	if !createProfileResponse.Success || len(createProfileResponse.Err) > 0 {
		err = fmt.Errorf(
			"invalid resposne during profile creation: %v, err: %v",
			response,
			createProfileResponse.Err,
		)

		return
	}

	log.Println("created profile, waiting for task response")

	err, response = client.GetTaskIDResponse(createProfileResponse.TaskID)

	return
}

func (client *Client) GetTaskIDResponse(taskID string) (
	err error,
	currentResponse *CreateProfileTaskResponse,
) {
	currentResponse = &CreateProfileTaskResponse{}
	taskIDUrl := fmt.Sprintf("%s/%s/%s", client.endpoint, "api/v1/task/", taskID)
	resp, err := http.Get(taskIDUrl)

	defer func() {
		_ = resp.Body.Close()
	}()

	if nil != err {
		return
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if nil != err {
		return
	}

	err = json.Unmarshal(bodyBytes, currentResponse)

	if nil != err {
		return
	}

	log.Println(currentResponse)

	const (
		createdStatus    = "CREATED"
		processingStatus = "PROCESSING"
		failedStatus     = "FAILED"
		completeStatus   = "COMPLETE"
	)

	if processingStatus == currentResponse.Status || createdStatus == currentResponse.Status {
		time.Sleep(time.Second)

		return client.GetTaskIDResponse(taskID)
	}

	if failedStatus == currentResponse.Status {
		err = fmt.Errorf("could not create profile, err: %s", currentResponse.Err)

		return
	}

	if completeStatus != currentResponse.Status {
		err = fmt.Errorf("unexpected status from relayer api, status: %s", currentResponse.Status)
		return
	}

	return
}

func (client *Client) CreateProfileAsync(
	profileJsonUrl string,
	erc725ControllerKey *common.MixedcaseAddress,
	email string,
) (
	err error,
	response *CreateProfileResponse,
) {
	type ProfileRequest struct {
		ProfileJsonUrl      string                   `json:"profileJsonUrl"`
		Salt                string                   `json:"salt"`
		Erc725ControllerKey *common.MixedcaseAddress `json:"erc725ControllerKey"`
		Email               string                   `json:"email"`
	}

	randInt := rand.Intn(10000000)
	encodedSalt := hexutil.EncodeUint64(uint64(randInt))

	for i := len(encodedSalt); i < 66; i++ {
		encodedSalt = fmt.Sprintf("%sf", encodedSalt)
	}

	profileRequest := ProfileRequest{
		ProfileJsonUrl:      profileJsonUrl,
		Salt:                encodedSalt,
		Erc725ControllerKey: erc725ControllerKey,
		Email:               email,
	}

	jsonBody, err := json.Marshal(&profileRequest)

	if nil != err {
		return
	}

	url := fmt.Sprintf("%s/%s", client.endpoint, "api/v1/universalprofile/lsp3")

	requestBuffer := bytes.NewBuffer(jsonBody)
	resp, err := http.Post(url, "application/json", requestBuffer)

	if nil != err {
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if nil != err {
		return
	}

	response = &CreateProfileResponse{}

	err = json.Unmarshal(bodyBytes, response)

	return
}
