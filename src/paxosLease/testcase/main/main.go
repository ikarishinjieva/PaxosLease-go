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
	runTestcase("../004_extend_lease.tc")
	runTestcase("../005_new_proposer_join_long-run_cluster.tc")
	runTestcase("../006_extend_lease_when_two_node_propose.tc")
	runTestcase("../007_network_brain_split.tc")
	runTestcase("../008_network_brain_split_2.tc")
	runTestcase("../009_give_up_lease.tc")
}

func runTestcase(path string) {
	fmt.Println("Running ", path)
	tc := testcase.NewTestcase(path)
	if err := tc.Start(); nil != err {
		fmt.Println("case failed :\n", err)
		fmt.Println("\t\tLog:\n\t", strings.Join(tc.GetLogs(), "\n\t\t"))
	}
}
