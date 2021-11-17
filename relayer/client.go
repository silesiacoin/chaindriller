package relayer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"io/ioutil"
	"math/rand"
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
	Err                 string         `json:"error,omitempty"`
}

func New(endpoint string) (client *Client) {
	client = &Client{endpoint: endpoint}
	return
}

func NewStaging() (client *Client) {
	client = New(StagingEndpoint)
	return
}

func (client *Client) CreateProfile(profileJsonUrl string, erc725ControllerKey *common.MixedcaseAddress, email string) (
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
