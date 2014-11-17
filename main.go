package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("heaverc")
)

const (
	version        = "0.1"
	startStopError = "Cannot start and stop container simultaneously (-ST given)"
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

	requestsChain := &Requests{}
	requestsChain.Params.ContainerName = containerName
	requestsChain.Params.Hostname = hostname
	requestsChain.Params.PoolId = poolname

	if args["-S"] != false {
		requestsChain.EnqueueStartRequest()
	}

	if args["-C"] != false {
		requestsChain.EnqueueCreateRequest()
	}

	if args["-T"] != false {
		requestsChain.EnqueueStopRequest()
	}

	if args["-D"] != false {
		requestsChain.EnqueueDeleteRequest()
	}

	if args["-L"] != false {
		requestsChain.EnqueueListRequest()
	}

	if args["-H"] != false {
		requestsChain.EnqueueListHostsRequest()
	}

	if args["--image"] != nil {
		requestsChain.SetImageParam(args["--image"].(string))
	}

	if args["--key"] != nil {
		requestsChain.SetKeyParam(args["--key"].(string))
	}

	if args["--raw-key"] != nil {
		requestsChain.SetRawKeyParam(args["--raw-key"].(string))
	}

	resChan := make(chan string)
	errChan := make(chan error)
	doneChan := make(chan int)

	err := checkArgs(args)
	if err != nil {
		fmt.Print(err)
		fmt.Print("\n")
		os.Exit(1)
	}

	go requestsChain.Run(resChan, errChan, doneChan)

	for {
		select {
		case r := <-resChan:
			fmt.Print(r)
			fmt.Print("\n")

		case err := <-errChan:
			fmt.Print(err)
			os.Exit(1)

		case <-doneChan:
			fmt.Print("OK\n")
			os.Exit(0)
		}
	}
}

func checkArgs(args map[string]interface{}) error {
	if args["-S"] != false && args["-T"] != false {
		return errors.New(startStopError)
	}

	return nil
}
