package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	TMP_FILE    = "tmp.out"
	RESULT_FILE = "result.out"
)

const (
	COVERAGE_MODE_SET    = "set"
	COVERAGE_MODE_ATOMIC = "atomic"
	COVERAGE_MODE_COUNT  = "count"
)

type TestResult int8

const (
	TEST_RES_PASSED TestResult = 1
	TEST_RES_FAILED TestResult = 2
	TEST_RES_EMPTY  TestResult = 3
)

type Packages []string

func (p *Packages) String() string {
	return fmt.Sprintf("%s", *p)
}

func (p *Packages) Set(s string) error {
	*p = append(*p, s)
	return nil
}

func getAllPkgs() ([]string, error) {
	cmd := exec.Command("go", "list", "./...")

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	pkgs := strings.Split(out.String(), "\n")
	return pkgs[:len(pkgs)-1], err
}

func testSinglePkg(pkgName string, resultFile *os.File, coverMode string) TestResult {
	profileOption := fmt.Sprintf("-coverprofile=%s", TMP_FILE)
	modeOption := fmt.Sprintf("-covermode=%s", coverMode)
	cmd := exec.Command("go", "test", profileOption, modeOption, pkgName)

	defer os.Remove(TMP_FILE)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		// test failed or error
		return TEST_RES_FAILED
	}

	rawData, err := ioutil.ReadFile(TMP_FILE)
	if err != nil {
		// no tests found for the package
		return TEST_RES_EMPTY
	}

	// split coverage mode and results
	tmpResult := bytes.SplitN(rawData, []byte("\n"), 2)
	if len(tmpResult) < 2 {
		// tmp file is empty
		return TEST_RES_EMPTY
	}

	if _, err = resultFile.Write(tmpResult[1]); err != nil {
		panic(fmt.Sprintf("cannot add coverage profile to result file: %s\n", err))
	}

	return TEST_RES_PASSED
}

func combinedCoverage() (float64, error) {
	cmd := exec.Command("go", "tool", "cover", fmt.Sprintf("-func=%s", RESULT_FILE))

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	res := strings.Split(out.String(), "\n")
	totalCoverageStr := res[len(res)-2]

	// parse string and return percent only
	re, err := regexp.Compile(`total:[ \t]+\([a-z]+\)[ \t]+(?P<Percent>[0-9]*\.[0-9]*)`)
	if err != nil {
		panic("cannot compile regexp pattern")
	}

	parsedResult := re.FindStringSubmatch(totalCoverageStr)
	if len(res) < 2 {
		fmt.Println("cannot parse coverage result")
		return 0, errors.New("coverage parsing failed")
	}

	coveragePercent, err := strconv.ParseFloat(parsedResult[1], 64)
	if err != nil {
		fmt.Printf("cannot convert coverage value %s\n", parsedResult[1])
		return 0, errors.New("converting coverage value failed")
	}

	return coveragePercent, nil
}

func CleanFiles() {
	os.Remove(RESULT_FILE)
	os.Remove(TMP_FILE)
}

func WriteCoverMode(resultFile *os.File, coverMode string) {
	coverModeHeader := fmt.Sprintf("mode: %s\n", coverMode)
	resultFile.Write([]byte(coverModeHeader))
}

func RunTests(pkgs, exPkgs Packages, resultFD *os.File, coverMode string, showProgress bool) (passed bool, empty int, total int) {
	testPipelineStatus := true
	emptyPkgs := 0
	totalTested := 0
	for i, pkgName := range pkgs {
		if strInSlice(pkgName, exPkgs) {
			continue
		}

		totalTested++

		if showProgress {
			fmt.Printf("Progress: %3d%%\r", int(100*float64(i)/float64(len(pkgs))))
		}

		switch testSinglePkg(pkgName, resultFD, coverMode) {
		case TEST_RES_PASSED:
			continue
		case TEST_RES_FAILED:
			testPipelineStatus = false
		case TEST_RES_EMPTY:
			emptyPkgs += 1
		}
	}

	return testPipelineStatus, emptyPkgs, totalTested
}

func PrintResults(testPipelinePassed bool, coverage float64, emptyPkgs int, totalPkgsTested int) {
	var status string
	if testPipelinePassed {
		status = "passed"
	} else {
		status = "failed"
	}

	pkgsWithTests := 100 * (1 - float64(emptyPkgs)/float64(totalPkgsTested))
	//fmt.Printf("status: %s, coverage: %.1f%%, packages tested: %d, packages without tests: %d\n", status, coverage, totalPkgsTested, emptyPkgs)
	fmt.Printf("status: %s, coverage: %.1f%%, packages with tests: %.1f%%\n", status, coverage, pkgsWithTests)
}

func main() {
	// todo choose coverage mode
	var excludedPkgs Packages
	var showProgress bool
	flag.Var(&excludedPkgs, "e", "exclude package from testing pipeline")
	flag.BoolVar(&showProgress, "p", false, "show current progress of test pipeline")
	flag.Parse()

	pkgs, err := getAllPkgs()
	if err != nil {
		fmt.Errorf("cannot get package list: %s\n", err)
		return
	}
	if len(pkgs) < 1 {
		fmt.Println("no Go packages found")
		return
	}

	CleanFiles()
	resultFD, err := os.OpenFile(RESULT_FILE, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		fmt.Printf("error: cannot create file for results: %s\n", err)
		return
	}

	// set coverage mode
	coverMode := COVERAGE_MODE_SET
	WriteCoverMode(resultFD, coverMode)

	// todo set signal handlers

	// run all tests
	testsPassed, emptyPkgs, totalTested := RunTests(pkgs, excludedPkgs, resultFD, coverMode, showProgress)

	// process combined coverage profile
	coverage, err := combinedCoverage()
	if err != nil {
		fmt.Printf("error: cannot process results: %s\n", err)
		return
	}

	// remove temp files
	resultFD.Close()
	CleanFiles()

	PrintResults(testsPassed, coverage, emptyPkgs, totalTested)
}

func strInSlice(str string, strSlice []string) bool {
	for _, s := range strSlice {
		if str == s {
			return true
		}
	}
	return false
}
