package relayer

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"net/http"
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
}

func New(endpoint string) (client *Client) {
	client = &Client{endpoint: endpoint}
	return
}

func NewStaging() (client *Client) {
	client = New(StagingEndpoint)
	return
}

func (client *Client) CreateProfile(profileJsonUrl string, erc725ControllerKey common.Address, email string) (
	err error,
	response *CreateProfileResponse,
) {
	type ProfileRequest struct {
		ProfileJsonUrl      string         `json:"profileJsonUrl"`
		Salt                string         `json:"salt"`
		Erc725ControllerKey common.Address `json:"erc725ControllerKey"`
		Email               string         `json:"email"`
	}

	var (
		randomBytes   []byte
		responseBytes []byte
	)
	randomNumber, err := rand.Read(randomBytes)

	if nil != err {
		return
	}

	encodedSalt := hexutil.EncodeUint64(uint64(randomNumber))

	profileRequest := &ProfileRequest{
		ProfileJsonUrl:      profileJsonUrl,
		Salt:                encodedSalt,
		Erc725ControllerKey: erc725ControllerKey,
		Email:               email,
	}

	jsonBody, err := json.Marshal(profileRequest)

	if nil != err {
		return
	}

	url := fmt.Sprintf("%s/%s", client.endpoint, "universalprofile/lsp3")

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))

	if nil != err {
		return
	}

	response = &CreateProfileResponse{}
	_, err = resp.Body.Read(responseBytes)

	if nil != err {
		return
	}

	err = json.Unmarshal(responseBytes, response)

	return
}
