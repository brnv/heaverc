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
	version           = "0.1"
	defaultConfigFile = "/etc/heaverc-ng/config.toml"
)

var usage = `heaverc, the heaverd-ng client

	Usage:
	heaverc -Cn <name> -i <image>... [--config <file_path>]
	        [--key <path>] [--raw-key <rsa_key>] [--pool <poolname>]
			[--dryrun]
	heaverc -Sn <name> [--config <file_path>] [--dryrun]
	heaverc -Tn <name> [--config <file_path>] [--dryrun]
	heaverc -Dn <name> [--config <file_path>] [--dryrun]
	heaverc -TDn <name> [--config <file_path>] [--dryrun]
	heaverc -L [--host <hostname>] [--config <file_path>]
	heaverc -H [--config <file_path>] [--dryrun]
	heaverc -P [--config <file_path>] [--dryrun]
	heaverc -h | --help

	Options:
	-h, --help                      Show this help.
	-C, --create                    Create container.
	-S, --start                     Start container.
	-T, --stop                      Stop container.
	-D, --destroy                   Destroy  container.
	-L, --list                      List containers.
	-H, --host-list                 List hosts.
	-P, --pool-list                 List pools.
	-N, --dryrun                    Don't touch anything. report what will be done.
	-n <name>, --name <name>        Name of container.
	-i <image>, --image  <image>    Image(s) for container.
	--host <hostname>               Host to operate on.
	--pool <poolname>               Pool to create container on.
	-k <key_path>, --key <key_path> Public ssh key (will be added to root's auhorized keys).
	--raw-key <rsa_key>             Public ssh key as string.
	--config <path>                 Configuration file [default: /etc/heaverc-ng/config.toml].
`

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		fmt.Print(err)
		fmt.Print("\n")
		os.Exit(1)
	}

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

	requestsChain.SetContainerName(containerName)
	requestsChain.SetHostname(hostname)
	requestsChain.SetPoolname(poolname)

	if args["--dryrun"] != false {
		requestsChain.SetDryrun(true)
	}

	if args["--config"] == nil {
		args["--config"] = defaultConfigFile
	}

	config, err := getConfig(string(args["--config"].(string)))
	if err != nil {
		log.Fatal(err)
	}

	apiUrl, err := config.GetString("api_url")
	if err != nil {
		log.Fatal(err)
	}

	requestsChain.SetApiUrl(apiUrl)

	if args["--create"] != false {
		images := []string{}
		key := ""
		rawkey := ""

		if args["--image"] != nil {
			images = args["--image"].([]string)
		}

		if args["--key"] != nil {
			key = args["--key"].(string)
		}

		if args["--raw-key"] != nil {
			rawkey = args["--raw-key"].(string)
		}

		requestsChain.Enqueue(createRequest{
			Images: images,
			Key:    key,
			Rawkey: rawkey,
		})
	}

	if args["--start"] != false {
		requestsChain.Enqueue(startRequest{})
	}

	if args["--stop"] != false {
		requestsChain.Enqueue(stopRequest{})
	}

	if args["--destroy"] != false {
		requestsChain.Enqueue(deleteRequest{})
	}

	if args["--list"] != false {
		if hostname == "" {
			requestsChain.Enqueue(listAllHostsContainersRequest{})
		} else {
			requestsChain.Enqueue(listOneHostContainersRequest{})
		}
	}

	if args["--host-list"] != false {
		requestsChain.Enqueue(listHostsRequest{})
	}

	if args["--pool-list"] != false {
		requestsChain.Enqueue(listPoolsRequest{})
	}

	err = checkArgs(args)
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
			fmt.Print(err)
			fmt.Print("\n")
			os.Exit(1)

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
