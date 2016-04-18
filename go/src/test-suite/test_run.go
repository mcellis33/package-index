package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// TestRun controls the current state of the test program.
type TestRun struct {
	ServerAddr       string
	ConcurrencyLevel int
	Unluckiness      int
	startedAt        time.Time
	waiting          sync.WaitGroup
}

// Fail fails the test
func (t *TestRun) Fail(reason string) {
	log.Printf("FAIL (took %v): %s", time.Since(t.startedAt), reason)
	os.Exit(0)
	os.Exit(1)
}

//Failf fails the test with a formatted message
func (t *TestRun) Failf(format string, a ...interface{}) {
	t.Fail(fmt.Sprintf(format, a...))
}

//Faile fails the test with the error as its message
func (t *TestRun) Faile(err error) {
	t.Failf("%v", err)
}

func debugf(format string, a ...interface{}) {
	if *debugMode {
		log.Printf(format, a...)
	}
}

//Run executes the test: remove, index, then remove again a large amount of packages
func (t *TestRun) Run() {
	log.Printf("test start: server %s, concurrency %d, unluckiness %d", t.ServerAddr, t.ConcurrencyLevel, t.Unluckiness)

	t.startedAt = time.Now()

	homebrewPackages, err := BrewToPackages(&AllPackages{})
	if err != nil {
		panic(fmt.Sprintf("Error parsing packages"))
	}

	segmentedPackages := SegmentListPackages(homebrewPackages.Packages, t.ConcurrencyLevel)

	log.Println("Step 1: Attempting to remove any previously installed packages (by failed test runs or whatever other reason)")
	clientCounter := 0
	concurrentBruteforceRemovesAllPackages(clientCounter, t, segmentedPackages)

	log.Println("Step 2: Index all packages by brute-force")
	clientCounter = clientCounter + t.ConcurrencyLevel
	concurrentBruteforceIndexesPackages(clientCounter, t, segmentedPackages)

	log.Println("Step 3: Verify if all packages were correctly indexed")
	clientCounter = clientCounter + t.ConcurrencyLevel
	concurrentverifyAllPackages(clientCounter, t, segmentedPackages, OK)

	log.Println("Step 4: Remove all installed packages")
	clientCounter = clientCounter + t.ConcurrencyLevel
	concurrentBruteforceRemovesAllPackages(clientCounter, t, segmentedPackages)

	log.Println("Step 5: Verify if all packages were correctly removed")
	clientCounter = clientCounter + t.ConcurrencyLevel
	concurrentverifyAllPackages(clientCounter, t, segmentedPackages, FAIL)

	log.Printf("test end: took %v", time.Since(t.startedAt))
}

//MakeTestRun returns a new instance of a test run.
func MakeTestRun(serverAddr string, concurrencyLevel int, unluckiness int) *TestRun {
	return &TestRun{
		ServerAddr:       serverAddr,
		ConcurrencyLevel: concurrencyLevel,
		Unluckiness:      unluckiness,
	}
}

func bruteforceIndexesPackages(client PackageIndexerClient, packages []*Package, changeOfBeingUnluckyInPercent int) error {
	totalPackages := len(packages)
	debugf("%s brute-forcing indexing of %d packages", client.Name(), totalPackages)
	for numPackagesInstalledThisItearion := 0; numPackagesInstalledThisItearion < totalPackages; {
		numPackagesInstalledThisItearion = 0
		for _, pkg := range packages {
			if shouldSomethingBadHappen(changeOfBeingUnluckyInPercent) {
				err := sendBrokenMessage(client)
				if err != nil {
					return err
				}
			}

			err := indexPackage(client, pkg, OK)

			if err == nil {
				numPackagesInstalledThisItearion = numPackagesInstalledThisItearion + 1
			}

		}
		debugf("%s reports %v/%v packages indexed", client.Name(), numPackagesInstalledThisItearion, totalPackages)
	}

	return nil
}

func indexPackage(client PackageIndexerClient, pkg *Package, expectedStatus ResponseCode) error {
	msg := MakeIndexMessage(pkg)
	responseCode, err := client.Send(msg)

	if err != nil {
		return fmt.Errorf("%s found error when sending message [%s]: %v", client.Name(), msg, err)
	}

	if responseCode != expectedStatus {
		return fmt.Errorf("%s found error when indexing  package [%s], that depends on [%#v]. Expected response to be [%s], got [%s]", client.Name(), pkg.Name, pkg.Dependencies, expectedStatus, responseCode)
	}

	return nil
}

func bruteforceRemovesAllPackages(client PackageIndexerClient, packages []*Package, changeOfBeingUnluckyInPercent int) error {
	totalPackages := len(packages)
	debugf("%s brute-forcing removal of %d packages", client.Name(), totalPackages)
	for installedPackages := totalPackages; installedPackages > 0; {
		installedPackages = totalPackages

		for _, pkg := range packages {
			msg := MakeRemoveMessage(pkg)
			responseCode, err := client.Send(msg)
			if err != nil {
				return fmt.Errorf("%s found error when sending message [%s]: %v", client.Name(), msg, err)
			}

			if responseCode == OK {
				installedPackages = installedPackages - 1
			}

		}
		debugf("%s reports %d/%d packages still installed", client.Name(), installedPackages, totalPackages)
	}
	return nil
}

func verifyAllPackages(client PackageIndexerClient, packages []*Package, expectedResponseCode ResponseCode, changeOfBeingUnluckyInPercent int) error {
	totalPackages := len(packages)
	debugf("%s querying for %d packages and expecting status code to be [%s]", client.Name(), totalPackages, expectedResponseCode)
	for _, pkg := range packages {
		msg := MakeQueryMessage(pkg)
		responseCode, err := client.Send(msg)
		if err != nil {
			return fmt.Errorf("%s found error when sending message [%s]: %v", client.Name(), msg, err)
		}

		if responseCode != expectedResponseCode {
			return fmt.Errorf("%s expected query for package [%s] to return [%s], got [%s]", client.Name(), pkg.Name, expectedResponseCode, responseCode)
		}
	}

	return nil
}

func makeClient(clientName string, t *TestRun) PackageIndexerClient {
	client, err := MakeTCPPackageIndexClient(clientName, t.ServerAddr)
	if err != nil {
		t.Failf("Error opening client to t.ServerAddr [%s]: %v", t.ServerAddr, err)
	}
	return client
}

func concurrentBruteforceIndexesPackages(clientCounter int, t *TestRun, segmentedPackages [][]*Package) {
	t.waiting.Add(t.ConcurrencyLevel)
	for _, p := range segmentedPackages {
		clientCounter++
		go func(number int, packagesToProcess []*Package) {
			name := fmt.Sprintf("client[%d]", number+1)
			debugf("Starting %s", name)
			defer t.waiting.Done()

			client := makeClient(name, t)
			defer client.Close()

			err := bruteforceIndexesPackages(client, packagesToProcess, t.Unluckiness)
			if err != nil {
				t.Failf("%v", err)
			}
		}(clientCounter, p)
	}
	t.waiting.Wait()
}

func concurrentBruteforceRemovesAllPackages(clientCounter int, t *TestRun, segmentedPackages [][]*Package) {
	t.waiting.Add(t.ConcurrencyLevel)
	for _, p := range segmentedPackages {
		clientCounter++
		go func(number int, packagesToProcess []*Package) {
			name := fmt.Sprintf("client[%d]", number+1)
			debugf("Starting %s", name)
			defer t.waiting.Done()

			client := makeClient(name, t)
			defer client.Close()

			err := bruteforceRemovesAllPackages(client, packagesToProcess, t.Unluckiness)
			if err != nil {
				t.Failf("%v", err)
			}
		}(clientCounter, p)
	}
	t.waiting.Wait()
}

func concurrentverifyAllPackages(clientCounter int, t *TestRun, segmentedPackages [][]*Package, expectedRepose ResponseCode) {
	t.waiting.Add(t.ConcurrencyLevel)
	for _, p := range segmentedPackages {
		clientCounter++
		go func(number int, packagesToProcess []*Package) {
			name := fmt.Sprintf("client[%d]", number+1)
			debugf("Starting %s", name)
			defer t.waiting.Done()

			client := makeClient(name, t)
			defer client.Close()

			err := verifyAllPackages(client, packagesToProcess, expectedRepose, t.Unluckiness)
			if err != nil {
				t.Failf("%v", err)
			}
		}(clientCounter, p)
	}
	t.waiting.Wait()
}

func durationInMillis(d time.Duration) int64 {
	return d.Nanoseconds() / int64(time.Millisecond)
}

func shouldSomethingBadHappen(changeOfBeingUnluckyInPercent int) bool {
	return rand.Intn(100) < changeOfBeingUnluckyInPercent
}

func sendBrokenMessage(client PackageIndexerClient) error {
	msg := MakeBrokenMessage()
	response, err := client.Send(msg)

	if err != nil {
		return fmt.Errorf("%s sent broken message [%s] and expected response code [ERROR], but an error was returned: %v", client.Name(), msg, err)
	}

	if response != ERROR {
		return fmt.Errorf("%s sent broken message [%s] and expected response code [ERROR] but got status code [%s]", client.Name(), msg, response)
	}
	return nil
}
