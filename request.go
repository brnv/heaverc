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
	"text/template"
)

var (
	ApiUrl        = "http://localhost:8081/"
	ContainerName = ""
	HostName      = ""
	PoolName      = ""
)

const (
	apiVersion = "v2"

	MessageContainerStarted   = "Container started"
	MessageContainerStopped   = "Container stopped"
	MessageContainerDestroyed = "Container destroyed"

	ErrorNoSuchContainer = "No such container"
)

type executor interface {
	Execute(dryRun bool) (string, error)
}

type Requests struct {
	Queue  []executor
	DryRun bool
}

type (
	createRequest struct {
		Images []string
		Key    string
		Rawkey string
	}
	startRequest                  struct{}
	stopRequest                   struct{}
	deleteRequest                 struct{}
	listAllHostsContainersRequest struct{}
	listOneHostContainersRequest  struct{}
	listHostsRequest              struct{}
	listPoolsRequest              struct{}
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
	Ips    map[string][]string
}

func (r *Requests) Enqueue(request executor) {
	r.Queue = append(r.Queue, request)
}

func (r *Requests) Run(callback func(string)) error {
	for _, request := range r.Queue {
		result, err := request.Execute(r.DryRun)
		if err != nil {
			return err
		}
		callback(result)
	}
	return nil
}

func (request createRequest) Execute(dryRun bool) (string, error) {
	key, err := request.getKey()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s%s/p/%s/%s", ApiUrl, apiVersion, PoolName, ContainerName)

	if dryRun {
		for _, image := range request.Images {
			url = url + fmt.Sprintf(" image=%v", image)
		}

		if request.Rawkey != "" {
			url = url + fmt.Sprintf(" key=%v", request.Rawkey)
		} else if request.Key != "" {
			url = url + fmt.Sprintf(" key=%v", request.Key)
		}

		return "POST " + url, nil
	}

	raw, err := rawResponse(
		url, "POST",
		map[string]interface{}{
			"image": request.Images,
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

	return fmt.Sprintf("Created container %s (on %s) with "+
		"addresses: %v", c.Name, c.Host, c.Ip), nil
}

func (request createRequest) getKey() (string, error) {
	if request.Rawkey != "" {
		return request.Rawkey, nil
	}

	if request.Key != "" {
		key, err := ioutil.ReadFile(request.Key)
		if err != nil {
			return "", err
		}
		return string(key), nil
	}

	return "", nil
}

func (request startRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/c/%s/start", ApiUrl, apiVersion, ContainerName)

	if dryRun {
		return "POST " + url, nil
	}

	resp, err := execute(url, "POST", nil)
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

func (request stopRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/c/%s/stop", ApiUrl, apiVersion, ContainerName)

	if dryRun {
		return "POST " + url, nil
	}

	resp, err := execute(url, "POST", nil)
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

func (request deleteRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/c/%s", ApiUrl, apiVersion, ContainerName)

	if dryRun {
		return "DELETE " + url, nil
	}

	resp, err := execute(url, "DELETE", nil)
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

func (request listAllHostsContainersRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/h", ApiUrl, apiVersion)

	if dryRun {
		return "GET " + url, nil
	}

	raw, err := rawResponse(url, "GET", nil)
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

	return justifyStringsToRight(list), nil
}

func (request listOneHostContainersRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/h/%s/stats", ApiUrl, apiVersion, HostName)

	if dryRun {
		return "GET " + url, nil
	}

	raw, err := rawResponse(url, "GET", nil)
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

	return justifyStringsToRight(list), nil
}

func (request listHostsRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/h", ApiUrl, apiVersion)

	if dryRun {
		return "GET " + url, nil
	}

	raw, err := rawResponse(url, "GET", nil)
	if err != nil {
		return "", err
	}

	hostsList := map[string]struct {
		Score        float64
		CpuCapacity  int64
		CpuUsage     int64
		RamCapacity  int64
		RamFree      int64
		DiskCapacity int64
		DiskFree     int64
		Containers   map[string]containerInfo
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

	const hostInfoTemplateDoc = `
{{.HostName}}
{{.Underline}}
score: {{.Score}}/1
cpu: {{.CpuFree}}/{{.CpuCapacity}} %
ram: {{.RamFree}}/{{.RamCapacity}} MiB
disk: {{.DiskFree}}/{{.DiskCapacity}} MiB
boxes:
{{.Containers}}
`
	hostInfoTemplate := template.Must(
		template.New("hostInfo").Parse(hostInfoTemplateDoc))

	type hostInfoTemplateStruct struct {
		HostName     string
		Underline    string
		Score        string
		CpuFree      string
		CpuCapacity  string
		RamFree      string
		RamCapacity  string
		DiskFree     string
		DiskCapacity string
		Containers   string
	}

	for _, hostname := range keys {
		chunk := bytes.NewBufferString("")

		hostInfoTemplate.Execute(chunk, hostInfoTemplateStruct{
			HostName:  hostname,
			Underline: strings.Repeat("-", len(hostname)),
			Score:     fmt.Sprintf("%.4f", hostsList[hostname].Score),
			CpuFree: fmt.Sprint(hostsList[hostname].CpuCapacity -
				hostsList[hostname].CpuUsage),
			CpuCapacity:  fmt.Sprint(hostsList[hostname].CpuCapacity),
			RamFree:      fmt.Sprint(hostsList[hostname].RamFree / 1024),
			RamCapacity:  fmt.Sprint(hostsList[hostname].RamCapacity / 1024),
			DiskFree:     fmt.Sprint(hostsList[hostname].DiskFree / 1024),
			DiskCapacity: fmt.Sprint(hostsList[hostname].DiskCapacity / 1024),
			Containers: justifyStringsToRight(
				getContainersStringedArray(
					hostsList[hostname].Containers)),
		})

		list = append(list, chunk.String())
	}

	return singleString(list), nil
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
				"%"+strconv.Itoa(maxNameLen)+"s (on %s): %8s, ip: %18s",
				containers[k].Name,
				containers[k].Host,
				containers[k].Status,
				containers[k].Ips["eth0"][0],
			))
	}

	return containersListStringed
}

func (request listPoolsRequest) Execute(dryRun bool) (string, error) {
	url := fmt.Sprintf("%s%s/h", ApiUrl, apiVersion)

	if dryRun {
		return "GET " + url, nil
	}

	raw, err := rawResponse(url, "GET", nil)
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

	return singleString(pools), nil
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

func justifyStringsToRight(strings []string) string {
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

func singleString(strings []string) string {
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
