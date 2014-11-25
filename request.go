package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
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

	MessageContainerStarted   = "Container started"
	MessageContainerStopped   = "Container stopped"
	MessageContainerDestroyed = "Container destroyed"

	ErrorNoSuchContainer = "No such container"
)

type executor interface {
	Execute() (string, error)
}

type Requests struct {
	queue         []executor
	containerName string
	poolname      string
	hostname      string
	apiUrl        string
	dryrun        bool
}

type (
	defaultRequest struct {
		method string
		url    string
	}
	createRequest struct {
		defaultRequest
		Image  string
		Key    string
		Rawkey string
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
	Error string
	Msg   json.RawMessage
}

type containerInfo struct {
	Name   string
	Host   string
	Status string
	Ip     string
}

func (r *Requests) SetContainerName(containerName string) {
	r.containerName = containerName
}

func (r *Requests) SetHostname(hostname string) {
	r.hostname = hostname
}

func (r *Requests) SetPoolname(poolname string) {
	r.poolname = poolname
}

func (r *Requests) SetDryrun(dryrun bool) {
	r.dryrun = true
}

func (r *Requests) SetApiUrl(apiUrl string) {
	r.apiUrl = apiUrl
}

func (r *Requests) Run(
	resChan chan string,
	errChan chan error,
	doneChan chan int,
) {
	for _, request := range r.queue {
		if r.dryrun == true {
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

func (r createRequest) Execute() (string, error) {
	key, err := r.getKey()
	if err != nil {
		return "", err
	}

	raw, err := rawResponse(
		r.url,
		r.method,
		map[string]interface{}{
			"image": []string{r.Image},
			"key":   key,
		},
	)

	if err != nil {
		return "", err
	}

	jsonResp := heaverdJsonResponse{}
	err = json.Unmarshal(raw, &jsonResp)
	if err != nil {
		return "", err
	}

	if jsonResp.Error != "" {
		return "", errors.New(jsonResp.Error)
	}

	c := containerInfo{}
	err = json.Unmarshal(jsonResp.Msg, &c)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Created container %v with "+
		"addresses: %v", c.Name, c.Ip), nil
}

func (r createRequest) getKey() (string, error) {
	if r.Rawkey != "" {
		return r.Rawkey, nil
	}

	if r.Key != "" {
		key, err := ioutil.ReadFile(r.Key)
		if err != nil {
			return "", err
		}
		return string(key), nil
	}

	return "", nil
}

func (r startRequest) Execute() (string, error) {
	resp, err := execute(r.url, r.method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return MessageContainerStarted, nil

	case 404:
		return "", errors.New(ErrorNoSuchContainer)
	}

	return "", nil
}

func (r stopRequest) Execute() (string, error) {
	resp, err := execute(r.url, r.method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return MessageContainerStopped, nil

	case 404:
		return "", errors.New(ErrorNoSuchContainer)
	}

	return "", nil
}

func (r deleteRequest) Execute() (string, error) {
	resp, err := execute(r.url, r.method, nil)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 204:
		return MessageContainerDestroyed, nil

	case 404:
		return "", errors.New(ErrorNoSuchContainer)

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

		return "", errors.New(jsonResp.Error)
	}

	return "", nil
}

func (r listAllHostsContainersRequest) Execute() (string, error) {
	raw, err := rawResponse(r.url, r.method, nil)
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

	keys := []string{}
	for k := range hostsList {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := []string{}

	for _, hostname := range keys {
		list = append(list, getContainersStringedArray(
			hostsList[hostname].Containers)...,
		)
	}

	return formatToRightJustifiedString(list), nil
}

func (r listOneHostContainersRequest) Execute() (string, error) {
	raw, err := rawResponse(r.url, r.method, nil)
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

	list := getContainersStringedArray(host.Containers)

	return formatToRightJustifiedString(list), nil
}

func (r listHostsRequest) Execute() (string, error) {
	raw, err := rawResponse(r.url, r.method, nil)
	if err != nil {
		return "", err
	}

	hostsList := map[string]struct {
		CpuCapacity int64
		RamCapacity int64
		Containers  map[string]containerInfo
	}{}
	err = json.Unmarshal(raw, &hostsList)
	if err != nil {
		return "", err
	}

	keys := []string{}
	for k := range hostsList {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := []string{}

	for _, hostname := range keys {
		list = append(list,
			fmt.Sprintf(
				"%s\n%s\nboxes:",
				hostname,
				strings.Repeat("-", len(hostname)),
			))

		list = append(list,
			formatToRightJustifiedString(
				getContainersStringedArray(
					hostsList[hostname].Containers),
			)+"\n")
	}

	return formatToSingleString(list), nil
}

func getContainersStringedArray(containers map[string]containerInfo) []string {
	keys := []string{}
	for k := range containers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxNameLen := 0
	for _, c := range containers {
		if len(c.Name) > maxNameLen {
			maxNameLen = len(c.Name)
		}
	}

	containersListStringed := []string{}
	for _, k := range keys {
		containersListStringed = append(containersListStringed,
			fmt.Sprintf(
				"%"+strconv.Itoa(maxNameLen)+"s (on %s): %8s, ip: %15s",
				containers[k].Name,
				containers[k].Host,
				containers[k].Status,
				containers[k].Ip,
			))
	}

	return containersListStringed
}

func (r listPoolsRequest) Execute() (string, error) {
	raw, err := rawResponse(r.url, r.method, nil)
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

	pools := []string{}

	for _, host := range hostsList {
		for _, p := range host.Pools {
			app := true
			for _, poolname := range pools {
				if poolname == p {
					app = false
				}
			}
			if app {
				pools = append(pools, p)
			}
		}
	}

	return formatToSingleString(pools), nil
}

func execute(
	url string,
	method string,
	params map[string]interface{},
) (*http.Response, error) {
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

func rawResponse(
	url string,
	method string,
	params map[string]interface{},
) ([]byte, error) {
	resp, err := execute(url, method, params)

	if err != nil {
		return []byte{}, err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return raw, err
}

func (r *Requests) Enqueue(request executor) {
	switch req := request.(type) {
	case createRequest:
		req.method = "POST"
		if r.poolname != "" {
			req.url = r.getUrl(apiCreateInsidePoolRequestUrl)
		} else {
			req.url = r.getUrl(apiCreateRequestUrl)
		}
		r.queue = append(r.queue, req)
	case startRequest:
		req.method = "POST"
		req.url = r.getUrl(apiStartRequestUrl)
		r.queue = append(r.queue, req)
	case stopRequest:
		req.method = "POST"
		req.url = r.getUrl(apiStopRequestUrl)
		r.queue = append(r.queue, req)
	case deleteRequest:
		req.method = "DELETE"
		req.url = r.getUrl(apiDeleteRequestUrl)
		r.queue = append(r.queue, req)
	case listAllHostsContainersRequest:
		req.method = "GET"
		req.url = r.getUrl(apiHostsInfoRequestUrl)
		r.queue = append(r.queue, req)
	case listOneHostContainersRequest:
		req.method = "GET"
		req.url = r.getUrl(apiOneHostInfoRequestUrl)
		r.queue = append(r.queue, req)
	case listHostsRequest:
		req.method = "GET"
		req.url = r.getUrl(apiHostsInfoRequestUrl)
		r.queue = append(r.queue, req)
	case listPoolsRequest:
		req.method = "GET"
		req.url = r.getUrl(apiHostsInfoRequestUrl)
		r.queue = append(r.queue, req)
	}
}

func (r *Requests) getUrl(url string) string {
	url = strings.Replace(url, ":cid", r.containerName, 1)
	url = strings.Replace(url, ":poolid", r.poolname, 1)
	url = strings.Replace(url, ":hid", r.hostname, 1)

	apiUrl := apiUrlDefault
	if r.apiUrl != "" {
		apiUrl = r.apiUrl
	}

	return apiUrl + apiVersion + url
}

func formatToRightJustifiedString(strings []string) string {
	maxNameLen := 0
	for _, s := range strings {
		if len(s) > maxNameLen {
			maxNameLen = len(s)
		}
	}

	res := ""
	for i, str := range strings {
		res += fmt.Sprintf("%"+strconv.Itoa(maxNameLen)+"s", str)
		if i < len(strings)-1 {
			res += "\n"
		}
	}

	return res
}

func formatToSingleString(strings []string) string {
	keys := []int{}
	for k := range strings {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	res := ""
	for i := range keys {
		res += strings[i]
		if i < len(strings)-1 {
			res += "\n"
		}
	}

	return res
}
