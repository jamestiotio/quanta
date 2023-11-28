package test_integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/disney/quanta/test"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

// Requirements: Consul must NOT be running on localhost:8500 we will start our own

// just one test:
// go test -v -run TestBasic ./test-integration/...

// We're going to leverage the fact that docker build is very fast if nothing has changed.
// The nodes are left up. To reset them go to docker desktop (or whatever) and delete the containers.

// Hint: run TestBasic first to make sure stuff is working.

type DockerNodesRunnerSuite struct {
	test.BaseDockerSuite

	// suite.Suite
	// state *test.ClusterLocalState
	// total test.SqlInfo

	// consulAddress string
	// proxyAddress  []string
}

func (suite *DockerNodesRunnerSuite) TestOne() { // just do the setup and teardown
	suite.EqualValues(suite.Total.ExpectedRowcount, suite.Total.ActualRowCount)
	suite.EqualValues(0, len(suite.Total.FailedChildren))
}

func (suite *DockerNodesRunnerSuite) SetupSuite() {

	suite.SetupDockerCluster(3, 2)

	// // TODO: move this all to a separate function?
	// var err error
	// var out string

	// somethingRestarted := false

	// proxyConnect := test.ProxyConnectStrings{}
	// proxyConnect.Host = "127.0.0.1"
	// proxyConnect.User = "MOLIG004"
	// proxyConnect.Password = ""
	// proxyConnect.Port = "4000"
	// proxyConnect.Database = "quanta"

	// suite.state = &test.ClusterLocalState{}
	// suite.state.ProxyConnect = &proxyConnect
	// suite.state.Db, err = suite.state.ProxyConnect.ProxyConnectConnect()
	// check(err)

	// // check if consul is running
	// if !test.IsConsuleRunning() {
	// 	test.Sh("docker network rm mynet")
	// 	test.Sh("docker network create -d bridge --subnet=172.20.0.0/16 mynet")
	// 	test.Sh("docker run -d -p 8500:8500 -p 8600:8600/udp --network mynet	--name=myConsul consul:1.10 agent -dev -ui -client=0.0.0.0")
	// }
	// // get the IP address of the consul container --format {{.NetworkSettings.Networks.mynet.IPAddress}}
	// out, err = test.Shell("docker inspect --format {{.NetworkSettings.Networks.mynet.IPAddress}} myConsul", "")
	// fmt.Println("docker inspect myConsul", out, err)
	// suite.consulAddress = strings.TrimSpace(out)

	// // check if there's a new build of the node image
	// out, err = test.Shell("docker inspect --format {{.Id}} node", "")
	// if err != nil {
	// 	fmt.Println("docker inspect node", err, out)
	// 	// check(err)
	// 	out = ""
	// } else {
	// 	fmt.Println("docker inspect node", out)
	// }

	// // build the image, as necessary
	// beforeSha := out
	// out, err = test.Shell("docker build -t node -f ../test/docker-nodes/Dockerfile ../", "")
	// _ = out
	// check(err)
	// out, err = test.Shell("docker inspect --format {{.Id}} node", "")
	// check(err)
	// imageChanged := out != beforeSha // if the sha changed, we need to restart the nodes
	// fmt.Println("imageChanged", imageChanged)

	// // check the nodes and see if we need to start/restart them
	// nodeToPprof := 1 // to set pprof on a node, set this to the index
	// for index := 0; index < 3; index++ {
	// 	i := fmt.Sprintf("%d", index)
	// 	// check node is running
	// 	out, err = test.Shell("docker exec q-node-"+i+" pwd", "")
	// 	itsUp := false
	// 	if err == nil {
	// 		itsUp = out == "/quanta\n"
	// 	}
	// 	if itsUp && imageChanged {
	// 		stopAndRemoveContainer("q-node-" + i)
	// 	}
	// 	if !itsUp || imageChanged { // start the node as necessary
	// 		// quanta-node is the entrypoint, node is the image
	// 		// q-node-0 ./data-dir 0.0.0.0 4000 are the args
	// 		// port := fmt.Sprintf("%d", 4010+index) // -p port + ":4000

	// 		somethingRestarted = true

	// 		pprofPortMap := ""
	// 		if index == nodeToPprof {
	// 			pprofPortMap = " -p 6060:6060"
	// 		}
	// 		options := "-d --network mynet" + pprofPortMap + " --name q-node-" + i + " -t node"
	// 		cmd := "docker run " + options + " quanta-node --consul-endpoint " + suite.consulAddress + ":8500  q-node-" + i + " ./data-dir 0.0.0.0 4000"
	// 		if index == nodeToPprof {
	// 			cmd += " --pprof true"
	// 		}
	// 		out, err := test.Shell(cmd, "")
	// 		// check(err)
	// 		fmt.Println("docker node command", cmd)
	// 		fmt.Println("docker run node", out, err)
	// 	}
	// }

	// // Wait for the nodes to come up
	// // test.WaitForStatusGreen("127.0.0.1:8500") // does this even work? Why not?
	// // fmt.Println("WaitForStatusGreen")
	// if somethingRestarted {
	// 	time.Sleep(10 * time.Second)
	// }

	// // check the PROXIES and see if we need to start/restart them
	// proxyToPprof := -1 // to set pprof on a node, set this to the index
	// for index := 0; index < len(suite.proxyAddress); index++ {
	// 	i := fmt.Sprintf("%d", index)
	// 	// check node is running, quanta-proxy
	// 	out, err = test.Shell("docker exec quanta-proxy-"+i+" pwd", "")
	// 	itsUp := false
	// 	if err == nil {
	// 		itsUp = out == "/quanta\n"
	// 	}
	// 	if itsUp && imageChanged {
	// 		stopAndRemoveContainer("quanta-proxy-" + i)
	// 	}
	// 	if !itsUp || imageChanged { // start the proxy as necessary
	// 		somethingRestarted = true
	// 		// quanta-proxy is the entrypoint, node is the image
	// 		// --consul-endpoint 172.20.0.2:8500 are the args
	// 		pprofPortMap := ""
	// 		if index == proxyToPprof {
	// 			pprofPortMap = " -p 6060:6060"
	// 		}
	// 		port := fmt.Sprintf("%d", 4000+index)
	// 		options := "-d -p " + port + ":4000" + pprofPortMap + " --network mynet --name quanta-proxy-" + i + " -t node"
	// 		cmd := "docker run " + options + " quanta-proxy --consul-endpoint " + suite.consulAddress + ":8500"
	// 		if index == proxyToPprof {
	// 			cmd += " --pprof true"
	// 		}
	// 		out, err := test.Shell(cmd, "")
	// 		// check(err)
	// 		fmt.Println("docker proxy command", cmd)
	// 		fmt.Println("docker run", out, err)
	// 	}
	// }

	// if somethingRestarted {
	// 	time.Sleep(10 * time.Second)
	// }
	// // tode check if the proxies are up

	// for index := 0; index < len(suite.proxyAddress); index++ {
	// 	istr := fmt.Sprintf("%d", index)
	// 	out, err = test.Shell("docker inspect --format {{.NetworkSettings.Networks.mynet.IPAddress}} quanta-proxy-"+istr, "")
	// 	fmt.Println("docker inspect quanta-proxy", out, err)
	// 	suite.proxyAddress[index] = strings.TrimSpace(out)
	// 	if suite.proxyAddress[index] == "" {
	// 		suite.Fail("FAIL proxyAddress is empty")
	// 		suite.Fail("FAIL proxyAddress is empty")
	// 		suite.Fail("FAIL proxyAddress is empty")
	// 	}
	// }
	// if somethingRestarted {
	// 	time.Sleep(5 * time.Second)
	// }
}

func (suite *DockerNodesRunnerSuite) TearDownSuite() {
	// leave the cluster running
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDockerNodesRunnerSuite(t *testing.T) {
	ourSuite := new(DockerNodesRunnerSuite)
	suite.Run(t, ourSuite)

	ourSuite.EqualValues(ourSuite.Total.ExpectedRowcount, ourSuite.Total.ActualRowCount)
	ourSuite.EqualValues(0, len(ourSuite.Total.FailedChildren))
}

// Run two sqlrunners hitting both proxies, forever. Turn your computer into a heater.
func (suite *DockerNodesRunnerSuite) TestBasicTorture() {

	fmt.Println("TestBasicTorture")

	test.StopAndRemoveContainer("basic_queries0")
	test.StopAndRemoveContainer("basic_queries1")

	cmd := "docker run --name basic_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries_load.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	// fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err

	var errGroup errgroup.Group
	for i := 0; i < 2; i++ {

		index := i
		istr := fmt.Sprintf("%d", index)
		errGroup.Go(func() error {
			test.StopAndRemoveContainer("basic_queries" + istr)

			time.Sleep(5 * time.Second)

			cmd = "docker run --name basic_queries" + istr + " -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries_body.sql"
			cmd += " -validate"
			cmd += " -repeats 1000"
			cmd += " -host " + suite.ProxyAddress[index] // this is the proxy
			cmd += " -consul " + suite.ConsulAddress + ":8500"
			cmd += " -user MOLIG004"
			cmd += " db quanta"
			cmd += " -log_level DEBUG"

			out, err = test.Shell(cmd, "")
			//fmt.Println("sqlrunner run", out, err)
			_ = out
			_ = err
			return err
		})
	}
	errGroup.Wait()
}

func (suite *DockerNodesRunnerSuite) TestJoinsTorture() {

	fmt.Println("TestJoinsTorture")

	test.StopAndRemoveContainer("join_queries0")
	test.StopAndRemoveContainer("join_queries1")

	cmd := "docker run --name join_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/joins_sql_load.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	// fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err

	var errGroup errgroup.Group
	for i := 0; i < 2; i++ {

		index := i
		istr := fmt.Sprintf("%d", index)
		errGroup.Go(func() error {
			test.StopAndRemoveContainer("join_queries" + istr)

			time.Sleep(5 * time.Second)

			cmd = "docker run --name join_queries" + istr + " -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/joins_sql_body.sql"
			cmd += " -validate"
			cmd += " -repeats 1000"
			cmd += " -host " + suite.ProxyAddress[index] // this is the proxy
			cmd += " -consul " + suite.ConsulAddress + ":8500"
			cmd += " -user MOLIG004"
			cmd += " db quanta"
			cmd += " -log_level DEBUG"

			out, err = test.Shell(cmd, "")
			//fmt.Println("sqlrunner run", out, err)
			_ = out
			_ = err
			return err
		})
	}
	errGroup.Wait()
}

// TestJoinsOneTwo runs the load and then runs the queries once.
func (suite *DockerNodesRunnerSuite) TestJoinsOneTwo() {

	time.Sleep(5 * time.Second)

	fmt.Println("TestJoinOneTwo")

	test.StopAndRemoveContainer("join_queries0")

	cmd := "docker run --name join_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/joins_sql_load.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	// fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err

	test.StopAndRemoveContainer("join_queries0")

	time.Sleep(5 * time.Second)

	cmd = "docker run --name join_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/joins_sql_body.sql"
	cmd += " -validate"
	cmd += " -repeats 1"
	cmd += " -host " + suite.proxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.consulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err = test.Shell(cmd, "")
	//fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err
}

// TestBasicOneTwo is same as TestBasic does the load first and then the queries - 10 times
func (suite *DockerNodesRunnerSuite) TestBasicOneTwo() {

	time.Sleep(5 * time.Second)

	fmt.Println("TestBasicOneTwo")

	test.StopAndRemoveContainer("basic_queries0")

	cmd := "docker run --name basic_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries_load.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	// fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err

	test.StopAndRemoveContainer("basic_queries0")

	time.Sleep(5 * time.Second)

	cmd = "docker run --name basic_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries_body.sql"
	cmd += " -validate"
	cmd += " -repeats 1000"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err = test.Shell(cmd, "")
	//fmt.Println("sqlrunner run", out, err)
	_ = out
	_ = err

}

func (suite *DockerNodesRunnerSuite) TestBasic() {

	fmt.Println("TestBasic")

	test.StopAndRemoveContainer("basic_queries0")

	cmd := "docker run --name basic_queries0 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[0] // this is the proxy
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	fmt.Println("sqlrunner run", out, err)
}

// TestBasicProxy1 is the same as TestBasic but uses proxy 1
func (suite *DockerNodesRunnerSuite) TestBasicProxy1() {

	fmt.Println("TestBasic p1")
	index := 1

	test.StopAndRemoveContainer("basic_queries1")

	cmd := "docker run --name basic_queries1 -w /quanta/sqlrunner --network mynet -t node sqlrunner -script_file ./sqlscripts/basic_queries.sql"
	cmd += " -validate"
	cmd += " -host " + suite.ProxyAddress[index] // this is the proxy #1
	cmd += " -consul " + suite.ConsulAddress + ":8500"
	cmd += " -user MOLIG004"
	cmd += " db quanta"
	cmd += " -log_level DEBUG"

	out, err := test.Shell(cmd, "")
	fmt.Println("sqlrunner run", out, err)
}

// XTestBasic doesn't work because ExecuteSqlFile calls AnalyzeRow which might invoke create table
// and CreateTable wants to contact the nodes directly.
func (suite *DockerNodesRunnerSuite) XTestBasic() { // this would be better if it worked

	fmt.Println("TestBasic")

	got := test.ExecuteSqlFile(suite.State, "../sqlrunner/sqlscripts/basic_queries.sql")

	test.MergeSqlInfo(&suite.Total, got)

	for _, child := range got.FailedChildren {
		fmt.Println("child failed", child.Statement)
	}

	suite.EqualValues(got.ExpectedRowcount, got.ActualRowCount)
	suite.EqualValues(0, len(got.FailedChildren))

	// FIXME: see: select avg(age) as avg_age from customers_qa where age > 55 and avg_age = 70 limit 1; in the file
}
