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

type Requests struct {
	Queue     []interface{}
	UrlParams struct {
		ContainerName string
		PoolId        string
		Hostname      string
	}
}

type (
	request struct {
		method string
		url    string
	}
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

type heaverdJsonResponse struct {
	Err string          `json:"error"`
	Msg json.RawMessage `json:"msg"`
}

type containerInfo struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status"`
	Ip     string `json:"ip"`
}

func (r *Requests) Run(
	resChan chan string,
	errChan chan error,
	doneChan chan int) {

	for _, request := range r.Queue {
		switch req := request.(type) {

		case *createRequest:
			key := r.getKey(req)
			response, err := r.performRequest(req.url,
				req.method,
				map[string]interface{}{
					"image": []string{req.image},
					"key":   key,
				})
			answerRaw, err := ioutil.ReadAll(response.Body)
			heaverdJsonResponse := heaverdJsonResponse{}
			err = json.Unmarshal(answerRaw, &heaverdJsonResponse)
			container := containerInfo{}
			err = json.Unmarshal(heaverdJsonResponse.Msg, &container)
			if heaverdJsonResponse.Err != "" {

				errChan <- errors.New(heaverdJsonResponse.Err)
			}
			createResultMessage := fmt.Sprintf("Created container %v with "+
				"addresses: %v", container.Name, container.Ip)
			resChan <- formatOutput([]string{createResultMessage})
			errChan <- err

		case startRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			switch response.StatusCode {
			case 204:
				resChan <- formatOutput([]string{
					fmt.Sprintf("Container %v started", r.UrlParams.ContainerName),
				})
				errChan <- err
			case 404:
				errChan <- errors.New(
					fmt.Sprintf("No such container: %v\n", r.UrlParams.ContainerName))
			}

		case stopRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			switch response.StatusCode {
			case 204:
				resChan <- formatOutput([]string{
					fmt.Sprintf("Container %v stopped", r.UrlParams.ContainerName),
				})
				errChan <- err
			case 404:
				errChan <- errors.New(
					fmt.Sprintf("No such container: %v\n", r.UrlParams.ContainerName))
			}

		case deleteRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			switch response.StatusCode {
			case 204:
				resChan <- formatOutput([]string{
					fmt.Sprintf("Container %v destroyed", r.UrlParams.ContainerName),
				})
				errChan <- err
			case 404:
				errChan <- errors.New(
					fmt.Sprintf("No such container: %v\n", r.UrlParams.ContainerName))
			case 409:
				answerRaw, _ := ioutil.ReadAll(response.Body)
				heaverdJsonResponse := heaverdJsonResponse{}
				_ = json.Unmarshal(answerRaw, &heaverdJsonResponse)
				errChan <- errors.New(heaverdJsonResponse.Err)
			}

		case listAllHostsContainersRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			hostsListRaw, err := ioutil.ReadAll(response.Body)
			hostsList := map[string]struct {
				Containers map[string]containerInfo
			}{}
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
			resChan <- formatOutput(containersListStringed)
			errChan <- err

		case listOneHostContainersRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			hostinfo, err := ioutil.ReadAll(response.Body)
			host := struct {
				Containers map[string]containerInfo
			}{}
			err = json.Unmarshal(hostinfo, &host)

			containersListStringed := []string{}
			for _, c := range host.Containers {
				containersListStringed = append(containersListStringed,
					fmt.Sprintf("%s: %s, ip: %s",
						c.Name,
						c.Status,
						c.Ip))
			}
			resChan <- formatOutput(containersListStringed)
			errChan <- err

		case listHostsRequest:
			response, err := r.performRequest(req.url, req.method, nil)
			hostsListRaw, err := ioutil.ReadAll(response.Body)
			hostsList := map[string]interface{}{}
			err = json.Unmarshal(hostsListRaw, &hostsList)

			hostsListStringed := []string{}
			for hostname, _ := range hostsList {
				hostsListStringed = append(hostsListStringed, hostname)
			}

			resChan <- formatOutput(hostsListStringed)
			errChan <- err
		}
	}

	doneChan <- 1
}

func (r *Requests) performRequest(
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

func (r *Requests) EnqueueCreateRequest() {
	request := &createRequest{}
	request.method = "POST"
	if r.UrlParams.PoolId != "" {
		request.url = r.getUrl(apiCreateInsidePoolRequestUrl)
	} else {
		request.url = r.getUrl(apiCreateRequestUrl)
	}
	r.Queue = append(r.Queue, request)
}

func (r *Requests) SetImageParam(image string) {
	for _, request := range r.Queue {
		if v, ok := request.(*createRequest); ok {
			v.image = image
		}
	}
}

func (r *Requests) SetKeyParam(key string) {
	for _, request := range r.Queue {
		if v, ok := request.(*createRequest); ok {
			v.key = key
		}
	}
}

func (r *Requests) SetRawKeyParam(rawkey string) {
	for _, request := range r.Queue {
		if v, ok := request.(*createRequest); ok {
			v.rawkey = rawkey
		}
	}
}

func (r *Requests) EnqueueStartRequest() {
	request := startRequest{}
	request.method = "POST"
	request.url = r.getUrl(apiStartRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) EnqueueStopRequest() {
	request := stopRequest{}
	request.method = "POST"
	request.url = r.getUrl(apiStopRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) EnqueueDeleteRequest() {
	request := deleteRequest{}
	request.method = "DELETE"
	request.url = r.getUrl(apiDeleteRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) EnqueueListRequest() {
	if r.UrlParams.Hostname == "" {
		r.enqueueAllHostsContainersListRequest()
	} else {
		r.enqueueOneHostContainersListRequest()
	}
}

func (r *Requests) enqueueAllHostsContainersListRequest() {
	request := listAllHostsContainersRequest{}
	request.method = "GET"
	request.url = r.getUrl(apiHostsInfoRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) enqueueOneHostContainersListRequest() {
	request := listOneHostContainersRequest{}
	request.method = "GET"
	request.url = r.getUrl(apiOneHostInfoRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) EnqueueListHostsRequest() {
	request := listHostsRequest{}
	request.method = "GET"
	request.url = r.getUrl(apiHostsInfoRequestUrl)
	r.Queue = append(r.Queue, request)
}

func (r *Requests) getUrl(url string) string {
	url = strings.Replace(url, ":cid", r.UrlParams.ContainerName, 1)
	url = strings.Replace(url, ":poolid", r.UrlParams.PoolId, 1)
	url = strings.Replace(url, ":hid", r.UrlParams.Hostname, 1)
	return apiBaseUrl + apiVersion + url
}

func (r *Requests) getKey(request *createRequest) string {
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
