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

type RestApi struct {
	RequestsQueue []interface{}
	ContainerName string
	PoolId        string
	HostName      string
}

type request struct {
	method string
	url    string
}

type createRequest struct {
	request
	image string
	key   string
}

type startRequest struct {
	request
}

type stopRequest struct {
	request
}

func (api *RestApi) Execute() string {
	for _, request := range api.RequestsQueue {
		log.Notice("%v", request)
	}
	return "done"
}

func (api *RestApi) EnqueueCreateRequest() {
	request := &createRequest{}
	request.method = "POST"
	request.url = api.getUrl(apiCreateRequestUrl)
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

func (api *RestApi) EnqueueStartRequest() {
	request := &startRequest{}
	request.method = "POST"
	request.url = api.getUrl(apiStartRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) EnqueueStopRequest() {
	request := &stopRequest{}
	request.method = "POST"
	request.url = api.getUrl(apiStopRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) getUrl(url string) string {
	url = strings.Replace(url, ":cid", api.ContainerName, 1)
	url = strings.Replace(url, ":poolid", api.PoolId, 1)
	return apiBaseUrl + apiPrefix + url
}
