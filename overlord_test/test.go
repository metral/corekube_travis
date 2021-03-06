package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/metral/corekube_travis/framework"
	"github.com/metral/goheat"
	"github.com/metral/goheat/util"
	"github.com/metral/goutils"
	"github.com/metral/overlord/lib"
)

func createGitCmdParam() string {
	travisPR := os.Getenv("TRAVIS_PULL_REQUEST")
	overlordRepoSlug := "metral/overlord"

	repoURL := fmt.Sprintf("https://github.com/%s", overlordRepoSlug)
	repo := strings.Split(overlordRepoSlug, "/")[1]
	cmd := ""

	switch travisPR {
	case "false": // false aka build commit
		travisBranch := os.Getenv("TRAVIS_BRANCH")
		travisCommit := os.Getenv("TRAVIS_COMMIT")
		c := []string{
			fmt.Sprintf("/usr/bin/git clone -b %s %s", travisBranch, repoURL),
			fmt.Sprintf("/usr/bin/git -C %s checkout -qf %s", repo, travisCommit),
		}
		cmd = strings.Join(c, " ; ")
	default: // PR number
		c := []string{
			fmt.Sprintf("/usr/bin/git clone %s", repoURL),
			fmt.Sprintf("/usr/bin/git -C %s fetch origin +refs/pull/%s/merge",
				repo, travisPR),
			fmt.Sprintf("/usr/bin/git -C %s checkout -qf FETCH_HEAD", repo),
		}
		cmd = strings.Join(c, " ; ")
	}

	return cmd
}

func nodeK8sCountTest(
	config *util.HeatConfig, details *util.StackDetails) string {

	d := *details
	msg := ""
	sleepDuration := 10 //seconds

	for {
		msg = "nodeK8sCountTest: "

		masterIP := util.ExtractIPFromStackDetails(d, "master_ip")
		expectedNodeCount, _ := strconv.Atoi(
			d.Stack.Parameters["kubernetes_minion_count"].(string))

		var nodesResult lib.KNodesCountResult
		endpoint := fmt.Sprintf("http://%s:%s", masterIP, lib.Conf.KubernetesAPIPort)
		masterAPIurl := fmt.Sprintf(
			"%s/api/%s/nodes", endpoint, lib.Conf.KubernetesAPIVersion)
		log.Printf("Retieving Kubernetes nodes from: %s", masterAPIurl)

		headers := map[string]string{
			"Content-Type": "application/json",
		}

		p := goutils.HttpRequestParams{
			HttpRequestType: "GET",
			Url:             masterAPIurl,
			Headers:         headers,
		}

		_, bodyBytes, _ := goutils.HttpCreateRequest(p)

		json.Unmarshal(bodyBytes, &nodesResult)
		nodesCount := len(nodesResult.Items)

		msg += fmt.Sprintf("ExpectedCount: %d, NodeCount: %d",
			expectedNodeCount, nodesCount)
		log.Printf(msg)

		if nodesCount == expectedNodeCount {
			return "Passed"
		}

		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}

	return "Failed"
}

func runTests(config *util.HeatConfig, details *util.StackDetails) {
	framework.StartTestTimeout(10, config, details, nodeK8sCountTest)
}

func main() {
	flag.Parse()

	params := map[string]string{
		"git_command": createGitCmdParam(),
	}
	log.Printf("========================================")
	log.Printf("%s", params)
	log.Printf("========================================")

	config, stackDetails := framework.BuildConfigAndCreateStack(&params)
	runTests(config, stackDetails)

	if *framework.DeleteStack {
		goheat.DeleteStack(config, stackDetails.Stack.Links[0].Href)
	}
}
