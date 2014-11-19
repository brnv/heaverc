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
	apiUrlDefault                 = "http://localhost:8081/"
	apiVersion                    = "v2"
	apiStartRequestUrl            = "/c/:cid/start"
	apiCreateRequestUrl           = "/c/:cid"
	apiCreateInsidePoolRequestUrl = "/p/:poolid/:cid"
	apiStopRequestUrl             = "/c/:cid/stop"
	apiDeleteRequestUrl           = "/c/:cid"
	apiHostsInfoRequestUrl        = "/h"
	apiOneHostInfoRequestUrl      = "/h/:hid/stats"
)

type executor interface {
	Execute() (string, error)
}

type Requests struct {
	queue  []executor
	params struct {
		containerName string
		poolname      string
		hostname      string
		apiUrl        string
		dryrun        bool
	}
}

type (
	defaultRequest struct {
		Method string
		Url    string
	}
	createRequest struct {
		defaultRequest
		image  string
		key    string
		rawkey string
	}
	startRequest                  defaultRequest
	stopRequest                   defaultRequest
	deleteRequest                 defaultRequest
	listAllHostsContainersRequest defaultRequest
	listOneHostContainersRequest  defaultRequest
	listHostsRequest              defaultRequest
	listPoolsRequest              defaultRequest
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

func (r *Requests) SetContainerName(containerName string) {
	r.params.containerName = containerName
}

func (r *Requests) SetHostname(hostname string) {
	r.params.hostname = hostname
}

func (r *Requests) SetPoolname(poolname string) {
	r.params.poolname = poolname
}

func (r *Requests) SetDryrun(dryrun bool) {
	r.params.dryrun = true
}

func (r *Requests) SetApiUrl(apiUrl string) {
	r.params.apiUrl = apiUrl
}

func (r *Requests) Run(
	resChan chan string,
	errChan chan error,
	doneChan chan int) {

	for _, request := range r.queue {
		if r.params.dryrun == true {
			resChan <- fmt.Sprintf("%s", request)
			continue
		}
		res, err := request.Execute()
		if err != nil {
			errChan <- err
			continue
		}
		resChan <- res
	}

	doneChan <- 1
}

func (r *createRequest) Execute() (string, error) {
	key, err := r.getKey()
	if err != nil {
		return "", err
	}

	resp, err := execute(r.Url,
		r.Method,
		map[string]interface{}{
			"image": []string{r.image},
			"key":   key,
		})
	if err != nil {
		return "", err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	jsonResp := heaverdJsonResponse{}
	err = json.Unmarshal(raw, &jsonResp)
	if err != nil {
		return "", err
	}

	if jsonResp.Err != "" {
		return "", errors.New(jsonResp.Err)
	}

	c := containerInfo{}
	err = json.Unmarshal(jsonResp.Msg, &c)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Created container %v with "+
		"addresses: %v", c.Name, c.Ip), nil
}

func (r *createRequest) getKey() (string, error) {
	if r.rawkey != "" {
		return r.rawkey, nil
	}

	if r.key != "" {
		key, err := ioutil.ReadFile(r.key)
		if err != nil {
			return "", err
		}
		return string(key), nil
	}

	return "", nil
}

func (r startRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return "Container started", nil

	case 404:
		return "", errors.New("No such container")
	}

	return "", nil
}

func (r stopRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return "Container stopped", nil

	case 404:
		return "", errors.New("No such container")
	}

	return "", nil
}

func (r deleteRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return "Container destroyed", nil

	case 404:
		return "", errors.New("No such container")

	case 409:
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		jsonResp := heaverdJsonResponse{}
		err = json.Unmarshal(raw, &jsonResp)
		if err != nil {
			return "", err
		}

		return "", errors.New(jsonResp.Err)
	}

	return "", nil
}

func (r listAllHostsContainersRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hostsList := map[string]struct {
		Containers map[string]containerInfo
	}{}
	err = json.Unmarshal(raw, &hostsList)
	if err != nil {
		return "", err
	}

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

	return formatToString(containersListStringed), nil
}

func (r listOneHostContainersRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	host := struct {
		Containers map[string]containerInfo
	}{}
	err = json.Unmarshal(raw, &host)
	if err != nil {
		return "", err
	}

	containersListStringed := []string{}
	for _, c := range host.Containers {
		containersListStringed = append(containersListStringed,
			fmt.Sprintf("%s: %s, ip: %s",
				c.Name,
				c.Status,
				c.Ip))
	}

	return formatToString(containersListStringed), nil
}

func (r listHostsRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hostsList := map[string]interface{}{}
	err = json.Unmarshal(raw, &hostsList)
	if err != nil {
		return "", err
	}

	hostsListStringed := []string{}
	for hostname, _ := range hostsList {
		hostsListStringed = append(hostsListStringed, hostname)
	}

	return formatToString(hostsListStringed), nil
}

func (r listPoolsRequest) Execute() (string, error) {
	resp, err := execute(r.Url, r.Method, nil)
	if err != nil {
		return "", err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hostsList := map[string]struct {
		Pools []string
	}{}
	err = json.Unmarshal(raw, &hostsList)
	if err != nil {
		return "", err
	}

	poolListStringed := []string{}
	for _, host := range hostsList {
		for _, p := range host.Pools {
			poolListStringed = append(poolListStringed,
				fmt.Sprintf("%v", p))
		}

	}

	return formatToString(poolListStringed), nil
}

func execute(
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
		return nil, nil
	}

	return nil, nil
}

func (r *Requests) EnqueueCreateRequest() {
	request := &createRequest{}
	request.Method = "POST"

	if r.params.poolname != "" {
		request.Url = r.getUrl(apiCreateInsidePoolRequestUrl)
	} else {
		request.Url = r.getUrl(apiCreateRequestUrl)
	}

	r.queue = append(r.queue, request)
}

func (r *Requests) SetImageParam(image string) {
	for _, request := range r.queue {
		if v, ok := request.(*createRequest); ok {
			v.image = image
		}
	}
}

func (r *Requests) SetKeyParam(key string) {
	for _, request := range r.queue {
		if v, ok := request.(*createRequest); ok {
			v.key = key
		}
	}
}

func (r *Requests) SetRawKeyParam(rawkey string) {
	for _, request := range r.queue {
		if v, ok := request.(*createRequest); ok {
			v.rawkey = rawkey
		}
	}
}

func (r *Requests) EnqueueStartRequest() {
	request := startRequest{}
	request.Method = "POST"
	request.Url = r.getUrl(apiStartRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) EnqueueStopRequest() {
	request := stopRequest{}
	request.Method = "POST"
	request.Url = r.getUrl(apiStopRequestUrl)

	r.queue = append(r.queue, request)
}

func (r *Requests) EnqueueDeleteRequest() {
	request := deleteRequest{}
	request.Method = "DELETE"
	request.Url = r.getUrl(apiDeleteRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) EnqueueListRequest() {
	if r.params.hostname == "" {
		r.enqueueAllHostsContainersListRequest()
	} else {
		r.enqueueOneHostContainersListRequest()
	}
}

func (r *Requests) enqueueAllHostsContainersListRequest() {
	request := listAllHostsContainersRequest{}
	request.Method = "GET"
	request.Url = r.getUrl(apiHostsInfoRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) enqueueOneHostContainersListRequest() {
	request := listOneHostContainersRequest{}
	request.Method = "GET"
	request.Url = r.getUrl(apiOneHostInfoRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) EnqueueListHostsRequest() {
	request := listHostsRequest{}
	request.Method = "GET"
	request.Url = r.getUrl(apiHostsInfoRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) EnqueueListPoolsRequest() {
	request := listPoolsRequest{}
	request.Method = "GET"
	request.Url = r.getUrl(apiHostsInfoRequestUrl)
	r.queue = append(r.queue, request)
}

func (r *Requests) getUrl(url string) string {
	url = strings.Replace(url, ":cid", r.params.containerName, 1)
	url = strings.Replace(url, ":poolid", r.params.poolname, 1)
	url = strings.Replace(url, ":hid", r.params.hostname, 1)

	apiUrl := apiUrlDefault
	if r.params.apiUrl != "" {
		apiUrl = r.params.apiUrl
	}

	return apiUrl + apiVersion + url
}

func formatToString(strings []string) string {
	res := ""
	for i, str := range strings {
		res += str
		if i < len(strings)-1 {
			res += "\n"
		}
	}

	return res
}
