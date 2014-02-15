package streambot

import(
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"bytes"
	"io/ioutil"
	"time"
	"net/url"
	"github.com/mbiermann/go-cluster"
)

func NewAPI(hosts []string, stats *Statter) (api *API, err error) {
	config := &cluster.ClusterConfig{Hosts:hosts}
	cluster, err := cluster.NewCluster(config)
	if err != nil {
		err = errors.New(fmt.Sprintf("Unexpected error when initializing Cluster client: %v", err))
		return
	}
	api = new(API)
	api.Stats = stats
	api.Cluster = cluster
	return
}

type API struct {
	Cluster *cluster.Cluster
	Stats 	*Statter
}

type CreateChannelRequest struct {
	Name 	string `json:"name"`
}

type CreateChannelResponse struct {
	Id 	string `json:"id"`
}

func(api *API) CreateChannel(Name string) (id string, err error) {
	api.Stats.Count("webservice.CreateChannel")
	url := url.URL{}
	url.Path = "/v1/channels"
	reqData := &CreateChannelRequest{Name}
	buf, err := json.Marshal(reqData)
	if err != nil {
		fatalFormat := "Unexpected error when marshalling CreateChannelRequest `%v` into JSON: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, reqData, err))
		return
	}
	req, err := http.NewRequest("PUT", url.String(), bytes.NewReader(buf))
	if err != nil {
		fatalFormat := "Unexpected error when creating POST request with JSON data `%s`: %v"
		err = errors.New(fmt.Sprintf(fatalFormat, string(buf), err))
		return
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := api.Cluster.Do(req)
	if err != nil {
		api.Stats.Count("webservice.errors.CreateChannel")
		fatalFormat := "Unexpected error on create channel request (%v) at URL `%v` into JSON: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, reqData, url.String(), err))
		return
	}
	// Close response body when reach end of function to free connection
	defer res.Body.Close()
	if res == nil {
		api.Stats.Count("webservice.empty.CreateChannel")
		fatalFormat := "Unexpected empty response (%v) to create channel request (%v) at URL `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res, reqData, url.String()))
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
	api.Stats.Count("webservice.CreateSubscription")
	url := url.URL{}
	url.Path = fmt.Sprintf("/v1/channels/%s/subscriptions", fromChannelId)
	reqData := &CreateSubscriptionRequest{toChannelId, time.Now().Unix()}
	buf, err := json.Marshal(reqData)
	if err != nil {
		fatalFormat := "Unexpected error when marshalling CreateChannelRequest `%v` into JSON: `%v`"
		return errors.New(fmt.Sprintf(fatalFormat, reqData, err))
	}
	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(buf))
	if err != nil {
		api.Stats.Count("webservice.errors.CreateSubscription")
		fatalFormat := "Unexpected error when creating PUT request with JSON data `%s`: %v"
		return errors.New(fmt.Sprintf(fatalFormat, string(buf), err))
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := api.Cluster.Do(req)
	if err != nil {
		fatalFormat := "Unexpected error when create PUT request with data `%s` for submission " +
		"on URL `%v`: `%v`"
		return errors.New(fmt.Sprintf(fatalFormat, string(buf), url.String(), err))
	}
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
	api.Stats.Count("webservice.GetSubscriptionsOfChannelWithId")
	url := url.URL{}
	url.Path = fmt.Sprintf("/v1/channels/%s/subscriptions", id)
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fatalFormat := "Unexpected error when creating GET request: %v"
		err = errors.New(fmt.Sprintf(fatalFormat, err))
	}
	res, err := api.Cluster.Do(req)
	if err != nil {
		api.Stats.Count("webservice.errors.GetSubscriptionsOfChannelWithId")
		fatalFormat := "Unexpected error on subscriptions request at URL `%v`: `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, url.String(), err))
		return
	}
	// Close response body when reach end of function to free connection
	defer res.Body.Close()
	if res == nil {
		api.Stats.Count("webservice.empty.GetSubscriptionsOfChannelWithId")
		fatalFormat := "Unexpected empty response (%v) to subscriptions request at URL `%v`"
		err = errors.New(fmt.Sprintf(fatalFormat, res, url.String()))
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