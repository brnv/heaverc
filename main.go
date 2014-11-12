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
	heaverc [-h] [-S] [-C] [-T] [-D]
		[-n NAME] [-i IMAGE] [-k KEY]
	heaverc [options]

	Options:
	-h|--help		Show this help.
	-S|--start		Start container.
	-C|--create		Create container.
	-T|--stop		Stop container.
	-D|--destroy		Destroy  container.
	-n NAME, --name NAME	Name of container.
	-i IMAGE, --image IMAGE	Image(s) for container.
	-k KEY, --key KEY	Public ssh key (will be added to root's auhorized keys)
`

func main() {
	args, _ := docopt.Parse(usage, nil, true, version, false)

	containerName := ""

	if args["--name"] != nil {
		containerName = args["--name"].(string)
	}

	api := &RestApi{
		ContainerName: containerName,
	}

	if args["-C"] != false {
		api.EnqueueCreateRequest()
	}

	if args["-S"] != false {
		api.EnqueueStartRequest()
	}

	if args["-T"] != false {
		api.EnqueueStopRequest()
	}

	if args["-D"] != false {
		api.EnqueueDeleteRequest()
	}

	if args["--image"] != nil {
		api.SetImageParam(args["--image"].(string))
	}

	if args["--key"] != nil {
		api.SetKeyParam(args["--key"].(string))
	}

	result := api.Execute()

	log.Notice("%v", result)
}
