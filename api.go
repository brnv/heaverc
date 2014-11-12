package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

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

type deleteRequest struct {
	request
}

func (api *RestApi) Execute() string {
	for _, request := range api.RequestsQueue {
		log.Notice("%v", request)
		switch req := request.(type) {
		case *createRequest:
			api.performRequest(req.url, req.method, map[string]interface{}{
				"image": []string{req.image},
				"key":   req.key,
			})
		case *startRequest:
			api.performRequest(req.url, req.method, nil)
		case *stopRequest:
			api.performRequest(req.url, req.method, nil)
		case *deleteRequest:
			api.performRequest(req.url, req.method, nil)
		}
	}
	return "done"
}

func (api *RestApi) performRequest(
	url string, method string, params map[string]interface{}) {

	switch method {
	case "POST":
		paramsEncoded, _ := json.Marshal(params)
		resp, err := http.Post(url, "", bytes.NewBuffer(paramsEncoded))
		log.Notice("%v", resp)
		log.Notice("%v", err)
	case "DELETE":
		req, err := http.NewRequest("DELETE", url, nil)
		resp, err := http.DefaultClient.Do(req)
		err = err
		log.Notice("%v", resp)
	}

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

func (api *RestApi) EnqueueDeleteRequest() {
	request := &deleteRequest{}
	request.method = "DELETE"
	request.url = api.getUrl(apiDeleteRequestUrl)
	api.RequestsQueue = append(api.RequestsQueue, request)
}

func (api *RestApi) getUrl(url string) string {
	url = strings.Replace(url, ":cid", api.ContainerName, 1)
	url = strings.Replace(url, ":poolid", api.PoolId, 1)
	return apiBaseUrl + apiPrefix + url
}
