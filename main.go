package main

import (
	"fmt"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
)

var (
	version = "0.1"
	log     = logging.MustGetLogger("heaverc")
)

var usage = `heaverc, the heaverd-ng client

	Usage:
	heaverc [-h] [-S] [-C] [-T] [-D] [-L]
		[-n NAME] [-i IMAGE] [--host HOST] [-k KEY]
		[--pool POOL]

	Options:
	-h|--help		Show this help.
	-S|--start		Start container.
	-C|--create		Create container.
	-T|--stop		Stop container.
	-D|--destroy		Destroy  container.
	-L|--list		List containers.
	-n NAME, --name NAME	Name of container.
	-i IMAGE, --image IMAGE	Image(s) for container.
	--host HOST		Host to operate on.
	--pool POOL		Pool to create container on.
	-k KEY, --key KEY	Public ssh key (will be added to root's auhorized keys)
`

func main() {
	args, _ := docopt.Parse(usage, nil, true, version, false)

	containerName := ""

	if args["--name"] != nil {
		containerName = args["--name"].(string)
	}

	hostname := ""

	if args["--host"] != nil {
		hostname = args["--host"].(string)
	}

	poolname := ""

	if args["--pool"] != nil {
		poolname = args["--pool"].(string)
	}

	api := &RestApi{
		ContainerName: containerName,
		Hostname:      hostname,
		PoolId:        poolname,
	}

	if args["-S"] != false {
		api.EnqueueStartRequest()
	}

	if args["-C"] != false {
		api.EnqueueCreateRequest()
	}

	if args["-T"] != false {
		api.EnqueueStopRequest()
	}

	if args["-D"] != false {
		api.EnqueueDeleteRequest()
	}

	if args["-L"] != false {
		api.EnqueueListRequest()
	}

	if args["--image"] != nil {
		api.SetImageParam(args["--image"].(string))
	}

	if args["--key"] != nil {
		api.SetKeyParam(args["--key"].(string))
	}

	result, _ := api.Execute()

	fmt.Print(result + "\tOK\n")
}
