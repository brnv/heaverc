package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
	"github.com/zazab/zhash"
)

var (
	log = logging.MustGetLogger("heaverc")
)

const (
	version = "0.1"
)

var usage = `heaverc, the heaverd-ng client

	Usage:
	heaverc [-h] [-S] [-C] [-T] [-D] [-L] [-H] [-P] [-N]
		[-n <name>] [-i <image>] [--host <hostname>] [-k <key_path>]
		[--raw-key <rsa_key>] [--pool <poolname>][--config=<path>]

	heaverc -h | --help
	heaverc -Cn <name> -i <image> -i ...
	heaverc -Cn <name> -i <image> -k <file_path>
	heaverc -Cn <name> -i <image> --pool <poolname>
	heaverc -Sn <name>
	heaverc -Tn <name>
	heaverc -TDn <name> -N
	heaverc -L --host <hostname>
	heaverc -H

	Options:
	-h|--help                       Show this help.
	-S|--start                      Start container.
	-C|--create                     Create container.
	-T|--stop                       Stop container.
	-D|--destroy                    Destroy  container.
	-L|--list                       List containers.
	-H|--host-list                  List hosts.
	-P|--pool-list                  List pools.
	-N|--dryrun                     Don't touch anything. report what will be done.
	-n <name>, --name <name>N       Name of container.
	-i <image>, --image  <image>    Image(s) for container.
	--host <hostname>               Host to operate on.
	--pool <poolname>               Pool to create container on.
	-k <key_path>, --key <key_path> Public ssh key (will be added to root's auhorized keys).
	--raw-key <rsa_key>             Public ssh key as string.
	--config=<path>                 Configuration file.
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
	requestsChain.Params.Poolname = poolname

	if args["-N"] != false {
		requestsChain.Params.DryRun = true
	}

	if args["--config"] != nil {
		config, err := getConfig(string(args["--config"].(string)))
		if err != nil {
			log.Fatal(err)
		}

		apiBaseUrl, err := config.GetString("api_url")
		if err != nil {
			log.Fatal(err)
		}

		requestsChain.Params.ApiBaseUrl = apiBaseUrl
	}

	if args["-C"] != false {
		requestsChain.EnqueueCreateRequest()
	}

	if args["-S"] != false {
		requestsChain.EnqueueStartRequest()
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

	if args["-P"] != false {
		requestsChain.EnqueueListPoolsRequest()
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

	err := checkArgs(args)
	if err != nil {
		log.Fatal(err)
	}

	resChan := make(chan string)
	errChan := make(chan error)
	doneChan := make(chan int)

	go requestsChain.Run(resChan, errChan, doneChan)

	for {
		select {
		case r := <-resChan:
			fmt.Print(r)
			fmt.Print("\n")

		case err := <-errChan:
			log.Fatal(err)

		case <-doneChan:
			fmt.Print("OK\n")
			os.Exit(0)
		}
	}
}

func checkArgs(args map[string]interface{}) error {
	return nil
}

func getConfig(path string) (zhash.Hash, error) {
	f, err := os.Open(path)
	if err != nil {
		return zhash.Hash{}, err
	}

	config := zhash.NewHash()
	config.ReadHash(bufio.NewReader(f))

	return config, nil
}
