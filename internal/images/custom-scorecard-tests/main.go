// Copyright The Cryostat Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	scapiv1alpha3 "github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"

	tests "github.com/cryostatio/cryostat-operator/internal/test/scorecard"
)

const podBundleRoot = "/bundle"

const argInstallOpenShiftCertManager = "installOpenShiftCertManager"

func main() {
	openShiftCertManager := flag.Bool(argInstallOpenShiftCertManager, false, "installs the cert-manager Operator for Red Hat OpenShift")
	flag.Parse()
	if openShiftCertManager == nil {
		// Default to false
		openShiftCertManager = &[]bool{false}[0]
	}

	entrypoint := flag.Args()
	if len(entrypoint) == 0 {
		log.Fatal("specify one or more test name arguments")
	}

	// Get namespace from SCORECARD_NAMESPACE environment variable
	namespace := os.Getenv("SCORECARD_NAMESPACE")
	if len(namespace) == 0 {
		log.Fatal("SCORECARD_NAMESPACE environment variable not set")
	}

	// Read the pod's untar'd bundle from a well-known path.
	bundle, err := apimanifests.GetBundleFromDir(podBundleRoot)
	if err != nil {
		log.Fatalf("failed to read bundle manifest: %s", err.Error())
	}

	var results []scapiv1alpha3.TestResult

	// Check that test arguments are valid
	if !validateTests(entrypoint) {
		results = printValidTests()
	} else {
		results = runTests(entrypoint, bundle, namespace, *openShiftCertManager)
	}

	// Print results in expected JSON form
	printJSONResults(results)
}

func printValidTests() []scapiv1alpha3.TestResult {
	result := scapiv1alpha3.TestResult{}
	result.State = scapiv1alpha3.FailState
	result.Errors = make([]string, 0)
	result.Suggestions = make([]string, 0)

	str := fmt.Sprintf("valid tests for this image include: %s", strings.Join([]string{
		tests.OperatorInstallTestName,
		tests.CryostatCRTestName,
	}, ","))
	result.Errors = append(result.Errors, str)

	return []scapiv1alpha3.TestResult{result}
}

func validateTests(testNames []string) bool {
	for _, testName := range testNames {
		switch testName {
		case tests.OperatorInstallTestName:
		case tests.CryostatCRTestName:
		default:
			return false
		}
	}
	return true
}

func runTests(testNames []string, bundle *apimanifests.Bundle, namespace string,
	openShiftCertManager bool) []scapiv1alpha3.TestResult {
	results := []scapiv1alpha3.TestResult{}

	// Run tests
	for _, testName := range testNames {
		switch testName {
		case tests.OperatorInstallTestName:
			results = append(results, tests.OperatorInstallTest(bundle, namespace))
		case tests.CryostatCRTestName:
			results = append(results, tests.CryostatCRTest(bundle, namespace, openShiftCertManager))
		default:
			log.Fatalf("unknown test found: %s", testName)
		}
	}
	return results
}

func printJSONResults(results []scapiv1alpha3.TestResult) {
	status := scapiv1alpha3.TestStatus{
		Results: results,
	}
	prettyJSON, err := json.MarshalIndent(status, "", "    ")
	if err != nil {
		log.Fatal("failed to generate json", err)
	}
	fmt.Printf("%s\n", string(prettyJSON))
}
