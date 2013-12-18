package main

import (
	"fmt"
	"paxosLease/testcase"
	"runtime"
	"strings"
)

func main() {
	runtime.GOMAXPROCS(50)
	runTestcase("../000_hello_world.tc")
	runTestcase("../001_two_node_propose.tc")
	runTestcase("../002_no_enough_nodes_response_prepare_request.tc")
	runTestcase("../003_node_got_lease_and_lease_expire.tc")
}

func runTestcase(path string) {
	fmt.Println("Running ", path)
	tc := testcase.NewTestcase(path)
	if err := tc.Start(); nil != err {
		fmt.Println("case failed :\n", err)
		fmt.Println("\t\tLog:\n\t", strings.Join(tc.GetLogs(), "\n\t\t"))
	}
}
