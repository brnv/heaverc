package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	//apiBaseUrl                    = "http://container.s:8081/"
	apiVersion                    = "v2"
	apiStartRequestUrl            = "/c/:cid/start"
	apiCreateRequestUrl           = "/c/:cid"
	apiCreateInsidePoolRequestUrl = "/p/:poolid/:cid"
	apiStopRequestUrl             = "/c/:cid/stop"
	apiDeleteRequestUrl           = "/c/:cid"
	apiHostsInfoRequestUrl        = "/h"
	apiOneHostInfoRequestUrl      = "/h/:hid/stats"
	apiBaseUrl                    = "http://lxbox.host.s:8081/"
)

type apiAnswer struct {
	Err string          `json:"error"`
	Msg json.RawMessage `json:"msg"`
}

type container struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status"`
	Ip     string `json:"ip"`
}

type host struct {
	Containers map[string]container
}

type hostsList map[string]struct {
	Containers map[string]container
}

type RestApi struct {
	RequestsQueue []interface{}
	ContainerName string
	PoolId        string
	Hostname      string
}

type request struct {
	method string
	url    string
}

type (
	createRequest struct {
		request
		image  string
		key    string
		rawkey string
	}
	startRequest                  struct{ request }
	stopRequest                   struct{ request }
	deleteRequest                 struct{ request }
	listAllHostsContainersRequest struct{ request }
	listOneHostContainersRequest  struct{ request }
	listHostsRequest              struct{ request }
)

func (api *RestApi) Execute() (string, error) {
	for _, request := range api.RequestsQueue {
		switch req := request.(type) {

		case *createRequest:
			key := api.getKey(req)
			response, err := api.performRequest(req.url,
				req.method,
				map[string]interface{}{
					"image": []string{req.image},
					"key":   key,
				})
			answerRaw, err := ioutil.ReadAll(response.Body)
			apiAnswer := apiAnswer{}
			err = json.Unmarshal(answerRaw, &apiAnswer)
			container := container{}
			err = json.Unmarshal(apiAnswer.Msg, &container)
			if apiAnswer.Err != "" {
				return "", errors.New(apiAnswer.Err)
			}
			createResultMessage := fmt.Sprintf("Created container %v with "+
				"addresses: %v", container.Name, container.Ip)
			return formatOutput([]string{createResultMessage}), err

		case startRequest:
			response, err := api.performRequest(req.url, req.method, nil)
			switch response.StatusCode {
			case 204:
				return formatOutput([]string{
					fmt.Sprintf("Container %v started", api.ContainerName),
				}), err
			case 404:
				return "", errors.New(
					fmt.Sprintf("No such container: %v", api.ContainerName))
			}

		case stopRequest:
			response, err := api.performRequest(req.url, req.method, nil)
			switch response.StatusCode {
			case 204:
				return formatOutput([]string{
					fmt.Sprintf("Container %v stopped", api.ContainerName),
				}), err
			case 404:
				return "", errors.New(
					fmt.Sprintf("No such container: %v", api.ContainerName))
			}

		case deleteRequest:
			api.performRequest(req.url, req.method, nil)

		case listAllHostsContainersRequest:
			response, err := api.performRequest(req.url, req.method, nil)
			hostsListRaw, err := ioutil.ReadAll(response.Body)
			hostsList := hostsList{}
			err = json.Unmarshal(hostsListRaw, &hostsList)

			containersListStringed := []string{}
			for _, host := range hostsList {
				for _, c := range host.Containers {
					containersListStringed = append(containersListStringed,
						fmt.Sprintf("%s: %s, ip: %s",
							c.Name,
							c.Status,
							c.Ip))
				}
			}
			return formatOutput(containersListStringed), err

		case listOneHostContainersRequest:
			response, err := api.performRequest(req.url, req.method, nil)
			hostinfo, err := ioutil.ReadAll(response.Body)
			host := host{}
			err = json.Unmarshal(hostinfo, &host)

			containersListStringed := []string{}
			for _, c := range host.Containers {
				containersListStringed = append(containersListStringed,
					fmt.Sprintf("%s: %s, ip: %s",
						c.Name,
						c.Status,
						c.Ip))
			}
			return formatOutput(containersListStringed), err

		case listHostsRequest:
			response, err := api.performRequest(req.url, req.method, nil)
			hostsListRaw, err := ioutil.ReadAll(response.Body)
			hostsList := hostsList{}
			err = json.Unmarshal(hostsListRaw, &hostsList)

			hostsListStringed := []string{}
			for hostname, _ := range hostsList {
				hostsListStringed = append(hostsListStringed, hostname)
			}
			return formatOutput(hostsListStringed), err
		}
	}
	return "", nil
}

func (api *RestApi) performRequest(
	url string,
	method string,
	params map[string]interface{}) (*http.Response, error) {

	switch method {
	case "GET":
		resp, err := http.Get(url)
		return resp, err
	case "POST":
		paramsEncoded, _ := json.Marshal(params)
		resp, err := http.Post(url, "", bytes.NewBuffer(paramsEncoded))
		return resp, err
	case "DELETE":
		req, err := http.NewRequest("DELETE", url, nil)
		resp, err := http.DefaultClient.Do(req)
		return resp, err
	default:
	}

	return nil, nil
}

func (api *RestApi) EnqueueCreateRequest() {
	request := &createRequest{}
	request.method = "POST"
	if api.PoolId != "" {
		request.url = api.getUrl(apiCreateInsidePoolRequestUrl)
	} else {
		request.url = api.getUrl(apiCreateRequestUrl)
	}
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) SetImageParam(image string) {
	for _, request := range api.RequestsQueue {
		if v, ok := request.(*createRequest); ok {
			v.image = image
		}
	}
}

func (api *RestApi) SetKeyParam(key string) {
	for _, request := range api.RequestsQueue {
		if v, ok := request.(*createRequest); ok {
			v.key = key
		}
	}
}

func (api *RestApi) SetRawKeyParam(rawkey string) {
	for _, request := range api.RequestsQueue {
		if v, ok := request.(*createRequest); ok {
			v.rawkey = rawkey
		}
	}
}

func (api *RestApi) EnqueueStartRequest() {
	request := startRequest{}
	request.method = "POST"
	request.url = api.getUrl(apiStartRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) EnqueueStopRequest() {
	request := stopRequest{}
	request.method = "POST"
	request.url = api.getUrl(apiStopRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) EnqueueDeleteRequest() {
	request := deleteRequest{}
	request.method = "DELETE"
	request.url = api.getUrl(apiDeleteRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) EnqueueListRequest() {
	if api.Hostname == "" {
		api.enqueueAllHostsContainersListRequest()
	} else {
		api.enqueueOneHostContainersListRequest()
	}
}

func (api *RestApi) enqueueAllHostsContainersListRequest() {
	request := listAllHostsContainersRequest{}
	request.method = "GET"
	request.url = api.getUrl(apiHostsInfoRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) enqueueOneHostContainersListRequest() {
	request := listOneHostContainersRequest{}
	request.method = "GET"
	request.url = api.getUrl(apiOneHostInfoRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) EnqueueListHostsRequest() {
	request := listHostsRequest{}
	request.method = "GET"
	request.url = api.getUrl(apiHostsInfoRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) getUrl(url string) string {
	url = strings.Replace(url, ":cid", api.ContainerName, 1)
	url = strings.Replace(url, ":poolid", api.PoolId, 1)
	url = strings.Replace(url, ":hid", api.Hostname, 1)
	return apiBaseUrl + apiVersion + url
}

func (api *RestApi) getKey(request *createRequest) string {
	if request.rawkey != "" {
		return request.rawkey
	}
	key, _ := ioutil.ReadFile(request.key)
	return string(key)
}

func formatOutput(strings []string) string {
	res := ""
	for i, str := range strings {
		res += str
		if i < len(strings)-1 {
			res += "\n"
		}
	}
	return res
}
