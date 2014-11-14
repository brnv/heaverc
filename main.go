package main

import (
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("heaverc")
)

const (
	version = "0.1"
)

var usage = `heaverc, the heaverd-ng client

	Usage:
	heaverc [-h] [-S] [-C] [-T] [-D] [-L] [-H]
		[-n NAME] [-i IMAGE] [--host HOST] [-k KEY]
		[--raw-key RAW_KEY] [--pool POOL]

	Options:
	-h|--help		Show this help.
	-S|--start		Start container.
	-C|--create		Create container.
	-T|--stop		Stop container.
	-D|--destroy		Destroy  container.
	-L|--list		List containers.
	-H|--host-list	List hosts.
	-n NAME, --name NAME	Name of container.
	-i IMAGE, --image IMAGE	Image(s) for container.
	--host HOST		Host to operate on.
	--pool POOL		Pool to create container on.
	-k KEY, --key KEY	Public ssh key (will be added to root's auhorized keys).
	--raw-key RAW_KEY	Public ssh key as string.
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

	if args["-H"] != false {
		api.EnqueueListHostsRequest()
	}

	if args["--image"] != nil {
		api.SetImageParam(args["--image"].(string))
	}

	if args["--key"] != nil {
		api.SetKeyParam(args["--key"].(string))
	}

	if args["--raw-key"] != nil {
		api.SetRawKeyParam(args["--raw-key"].(string))
	}

	resChan := make(chan string)
	errChan := make(chan error)
	doneChan := make(chan int)
	go api.Execute(resChan, errChan, doneChan)

	for {
		select {
		case r := <-resChan:
			fmt.Print(r)
			fmt.Print("\n")
		case err := <-errChan:
			if err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
		case <-doneChan:
			fmt.Print("OK\n")
			os.Exit(0)
		}
	}
}
