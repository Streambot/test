package streambot

import(
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"bytes"
	"io/ioutil"
	"time"
)

func NewAPI(host string) *API {
	api := new(API)
	api.Host = host
	return api
}

type API struct {
	Host 	string
}

type CreateChannelRequest struct {
	Name 	string `json:"name"`
}

type CreateChannelResponse struct {
	Id 	string `json:"id"`
}

func(api *API) CreateChannel(Name string) (id string, err error) {
	url := fmt.Sprintf("http://%s/v1/channels", api.Host)
	reqData := &CreateChannelRequest{Name}
	buf, err := json.Marshal(reqData)
	if err != nil {
		fatalFormat := "Unexpected error when marshalling CreateChannelRequest `%v` into JSON: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, reqData, err))
		return
	}
	res, err := http.Post(url, "application/json", bytes.NewReader(buf))
	// Close response body when reach end of function to free connection
	defer res.Body.Close()
	if err != nil {
		fatalFormat := "Unexpected error on create channel request (%v) at URL `%v` into JSON: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, reqData, url, err))
		return
	}
	if res == nil {
		fatalFormat := "Unexpected empty response (%v) to create channel request (%v) at URL `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res, reqData, url))
		return	
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fatalFormat := "Unexpected error when read in create channel request response (%v): `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res.Body, err))
		return
	}
	var response CreateChannelResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fatalFormat := "Unexpected error when read create channel request response `%v` expected " +
		"to be JSON for CreateChannelResponse: %v"
		err = errors.New(fmt.Sprintf(fatalFormat, res.Body, err))
		return
	}
	id = response.Id
	return
}

type CreateSubscriptionRequest struct {
	ChannelId 	string `json:"channel_id"`
	Time		int64  `json:"created_at"`
}

type CreateSubscriptionResponse struct {
	Id 	string `json:"id"`
}

func(api *API) CreateSubscription(fromChannelId string, toChannelId string) error {
	url := fmt.Sprintf("http://%s/v1/channels/%s/subscriptions", api.Host, fromChannelId)
	reqData := &CreateSubscriptionRequest{toChannelId, time.Now().Unix()}
	buf, err := json.Marshal(reqData)
	if err != nil {
		fatalFormat := "Unexpected error when marshalling CreateChannelRequest `%v` into JSON: `%v`"
		return errors.New(fmt.Sprintf(fatalFormat, reqData, err))
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(buf))
	if err != nil {
		fatalFormat := "Unexpected error when create PUT request with data `%s` for submission " +
		"on URL `%v`: `%v`"
		return errors.New(fmt.Sprintf(fatalFormat, string(buf), url, err))
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	// Close response body when reach end of function to free connection
	defer res.Body.Close()
	if err != nil {
		fatalFormat := "Unexpected error on create channel request (%v) at URL `%v` into JSON: `%v`"
		return errors.New(fmt.Sprintf(fatalFormat, reqData, url, err))
	}
	return nil
}

type Subscription struct {
	Id 		string `json:"id"`
	Name 	string `json:"name"`
}

func(api *API) GetSubscriptionsOfChannelWithId(id string) (channelIds []string, err error) {
	url := fmt.Sprintf("http://%s/v1/channels/%s/subscriptions", api.Host, id)
	fmt.Println(url)
	res, err := http.Get(url)
	// Close response body when reach end of function to free connection
	defer res.Body.Close()
	if err != nil {
		fatalFormat := "Unexpected error on subscriptions request at URL `%v`: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, url, err))
	}
	if res == nil {
		fatalFormat := "Unexpected empty response (%v) to subscriptions request at URL `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res, url))
		return	
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fatalFormat := "Unexpected error when read in subscriptions request response (%v): `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res.Body, err))
		return
	}
	var response []Subscription
	fmt.Println(string(body))
	err = json.Unmarshal(body, &response)
	if err != nil {
		fatalFormat := "Unexpected error when read subscriptions response `%v` expected " +
		"to be JSON for []Subscription: %v"
		err = errors.New(fmt.Sprintf(fatalFormat, res.Body, err))
		return
	}
	channelIds = []string{}
	for _, entry := range response {
		channelIds = append(channelIds, entry.Id)
	}
	return
}