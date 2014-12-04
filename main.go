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

const version = "0.1"

var usage = `heaverc, the heaverd-ng client

	http://git.rn/projects/DEVOPS/repos/heaverd-ng/

	Usage:
	heaverc -Cn <name> -p <poolname> -i <image>...
	        [--config <file_path>] [--key <key_path>]
	        [--raw-key <rsa_key>] [--dryrun]
	heaverc -Sn <name> [--config <file_path>] [--dryrun]
	heaverc -Tn <name> [--config <file_path>] [--dryrun]
	heaverc -Dn <name> [--config <file_path>] [--dryrun]
	heaverc -TDn <name> [--config <file_path>] [--dryrun]
	heaverc -L [--host <hostname>] [--config <file_path>] [--dryrun]
	heaverc -H [--config <file_path>] [--dryrun]
	heaverc -I [--config <file_path>] [--dryrun]
	heaverc -h | --help

	Options:
	-h, --help                       Show this help.
	-C, --create                     Create container.
	-S, --start                      Start container.
	-T, --stop                       Stop container.
	-D, --destroy                    Destroy  container.
	-L, --list                       List containers.
	-H, --host-list                  List hosts.
	-I, --pool-list                  List pools.
	-d, --dryrun                     Don't touch anything. Report what will be done.
	-n <name>, --name <name>         Name of container.
	-p <poolname>, --pool <poolname> Pool to create container on.
	-i <image>, --image <image>      Image(s) for container.
	--host <hostname>                Host to operate on.
	-k <key_path>, --key <key_path>  Public ssh key (will be added to root's auhorized keys).
	--raw-key <rsa_key>              Public ssh key as string.
	--config <file_path>             Configuration file [default: /etc/heaverc-ng/config.toml].
`

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	if args["--name"] != nil {
		ContainerName = args["--name"].(string)
	}

	if args["--pool"] != nil {
		PoolName = args["--pool"].(string)
	}

	if args["--host"] != nil {
		HostName = args["--host"].(string)
	}

	requestsChain := &Requests{}

	requestsChain.DryRun = args["--dryrun"].(bool)

	config, err := getConfig(string(args["--config"].(string)))
	if err != nil {
		log.Fatal(err)
	}

	ApiUrl, err = config.GetString("api_url")
	if err != nil {
		log.Fatal(err)
	}

	if args["--create"].(bool) {
		images := []string{}
		images = args["--image"].([]string)

		keyPath := ""
		if args["--key"] != nil {
			keyPath = args["--key"].(string)
		}

		rawkey := ""
		if args["--raw-key"] != nil {
			rawkey = args["--raw-key"].(string)
		}

		requestsChain.Enqueue(createRequest{
			Images: images,
			Key:    keyPath,
			Rawkey: rawkey,
		})
	}

	if args["--start"].(bool) {
		requestsChain.Enqueue(startRequest{})
	}

	if args["--stop"].(bool) {
		requestsChain.Enqueue(stopRequest{})
	}

	if args["--destroy"].(bool) {
		requestsChain.Enqueue(deleteRequest{})
	}

	if args["--list"].(bool) {
		if args["--host"] == nil {
			requestsChain.Enqueue(listAllHostsContainersRequest{})
		} else {
			requestsChain.Enqueue(listOneHostContainersRequest{})
		}
	}

	if args["--host-list"].(bool) {
		requestsChain.Enqueue(listHostsRequest{})
	}

	if args["--pool-list"].(bool) {
		requestsChain.Enqueue(listPoolsRequest{})
	}

	resultsCallback := func(result string) {
		fmt.Print(result + "\n")
	}

	err = requestsChain.Run(resultsCallback)

	if err != nil {
		fmt.Print(err.Error() + "\n")
		os.Exit(1)
	}

	fmt.Print("OK\n")
}

func getConfig(path string) (zhash.Hash, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return zhash.Hash{}, err
	}

	config := zhash.NewHash()
	config.ReadHash(bufio.NewReader(configFile))

	return config, nil
}
