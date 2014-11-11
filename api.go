package main

import "strings"

const (
	//apiBaseUrl                    = "http://container.s:8081/"
	apiPrefix                     = "v2"
	apiCreateRequestUrl           = "/c/:cid"
	apiCreateInsidePoolRequestUrl = "/p/:poolid/:cid"
	apiStartRequestUrl            = "/c/:cid/start"
	apiStopRequestUrl             = "/c/:cid/stop"
	apiDeleteRequestUrl           = "/c/:cid"
	apiHostsInfoRequestUrl        = "/h"
	apiBaseUrl                    = "http://lxbox.host.s:8081/"
)

type Api struct {
	containerName string
	poolId        string
	Operations    []*request
	Params        RequestParams
}

type RequestParams struct {
	image string
	key   string
}

type request struct {
	method string
	url    string
}

func getApiUrl(url string) string {
	return apiBaseUrl + apiPrefix + url
}

func (api *Api) SetContainerName(name string) {
	api.containerName = name
}

func (api *Api) SetParamKey(key string) {
	api.Params.key = key
}

func (api *Api) SetParamImage(image string) {
	api.Params.image = image
}

func (api *Api) AddCreateRequest() {
	request := &request{
		method: "POST",
		url:    getApiUrl(apiCreateRequestUrl),
	}
	api.Operations = append(api.Operations, request)
}

func (api *Api) AddStartRequest() {
	request := &request{
		method: "POST",
		url:    getApiUrl(apiStartRequestUrl),
	}
	api.Operations = append(api.Operations, request)
}

func (api *Api) AddStopRequest() {
	request := &request{
		method: "POST",
		url:    getApiUrl(apiStopRequestUrl),
	}
	api.Operations = append(api.Operations, request)
}

func (api *Api) Execute() string {
	api.replaceUrlPlaceholders()
	for _, request := range api.Operations {
		log.Notice("%v", request)
	}
	log.Notice("%v", api.Params)
	return "done"
}

func (api *Api) replaceUrlPlaceholders() {
	for _, request := range api.Operations {
		request.url = strings.Replace(request.url, ":cid", api.containerName, 1)
		request.url = strings.Replace(request.url, ":poolid", api.poolId, 1)
	}
}
