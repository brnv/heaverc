package main

import (
	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
)

var (
	version = "0.1"
	log     = logging.MustGetLogger("heaverc")
)

var usage = `heaverc, the heaverd-ng client

	Usage:
	heaverc [-S] [-C] [-T] [-n NAME] [-i IMAGE] [-k KEY]
	heaverc [options]

	Options:
	-h|--help		Show this help.
	-C|--create		Create container.
	-S|--start		Start container.
	-n NAME, --name NAME	Name of container.
	-i IMAGE, --image IMAGE	Image(s) for container.
	-k KEY, --key KEY	Public ssh key (will be added to root's auhorized keys)
`

func main() {
	args, _ := docopt.Parse(usage, nil, true, version, false)

	api := &Api{
		Params: RequestParams{},
	}

	if args["-C"] != false {
		api.AddCreateRequest()
	}

	if args["-S"] != false {
		api.AddStartRequest()
	}

	if args["-T"] != false {
		api.AddStopRequest()
	}

	if args["-n"] != nil {
		api.SetContainerName(args["-n"].(string))
	}

	if args["--name"] != nil {
		api.SetContainerName(args["--name"].(string))
	}

	if args["-i"] != nil {
		api.SetParamImage(args["-i"].(string))
	}

	if args["--image"] != nil {
		api.SetParamImage(args["--image"].(string))
	}

	if args["-k"] != nil {
		api.SetParamKey(args["-k"].(string))
	}

	if args["--key"] != nil {
		api.SetParamKey(args["--key"].(string))
	}

	result := api.Execute()
	log.Notice("%v", result)
}
