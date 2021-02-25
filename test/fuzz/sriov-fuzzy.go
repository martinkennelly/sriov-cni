package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/plugins/pkg/testutils"
)

// Default values
const (
	defPath      = "/opt/cni/bin/sriov"
	defTestNum   = 100000
	defPanicOnly = false
)

type cmdStatus struct {
	Type       string
	Successful bool
	Code       int
}

// This will call CNI specified in path, with selected command (tested for SRIOV-CNI with ADD/DEL)
// args will hold CNI arguments - the StdinData and other values that will be used as ENV variables
// Output will be logged to files provided with logfiles
// If panicOnly == true then only lines that contains 'panic' will be
// logged (not sure if this will work as expected as I am unable to cause panic)
func callCni(path, command string, args *skel.CmdArgs, logfile *os.File, panicOnly bool) (cmdStatus, error) {
	var result cmdStatus
	result.Type = command
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), "CNI_COMMAND="+command, "CNI_CONTAINERID="+args.ContainerID,
		"CNI_NETNS="+args.Netns, "CNI_IFNAME="+args.IfName, "CNI_PATH="+args.Path)
	cmd.Stdin = strings.NewReader(string(args.StdinData))
	output, err := cmd.CombinedOutput()
	strOutput := string(output)

	// Recover error code from output if available
	re := regexp.MustCompile(`"code": [0-9]*`)
	found := re.Find(output)
	if found == nil {
		result.Code = 0
	} else {
		strFound := string(found)
		strCode := strings.Fields(strFound)[1]
		result.Code, _ = strconv.Atoi(strCode)
	}

	// Log entries
	if err != nil {
		strErr := err.Error()
		result.Successful = false

		if (panicOnly && (strings.Contains(strings.ToLower(strErr), "panic") ||
			strings.Contains(strings.ToLower(strOutput), "panic"))) || !panicOnly {
			fmt.Fprintln(logfile, "Command: "+command+" - FAIL")
			fmt.Fprintln(logfile, "Error: "+err.Error())
			fmt.Fprintln(logfile, "Input data: "+string(args.StdinData))
			fmt.Fprintln(logfile, "Output: "+strOutput+"\n")
		}
		return result, err
	}

	result.Successful = true
	fmt.Fprintln(logfile, "Command: "+command+" - SUCCESS")
	fmt.Fprintln(logfile, "Input data: "+string(args.StdinData))
	fmt.Fprintln(logfile, "Output: "+strOutput+"\n")

	return result, nil
}

func printSummary(sCalls, fCalls *[]cmdStatus, logPath string) {
	ns := len(*sCalls)
	nf := len(*fCalls)
	fmt.Println("Performed", ns+nf, "tests of which:")
	fmt.Println(nf, "failed")
	fmt.Println(ns, "succeeded")
	fmt.Println("")

	var errCodes = make(map[int]int)
	for _, r := range *fCalls {
		errCodes[r.Code]++
	}

	fmt.Println("Errors by error code:")
	for k, v := range errCodes {
		fmt.Println("Code: ", k, "\tErrors:\t", v)
	}
	fmt.Println("\nMore details can be found in", logPath)
}

func main() {
	// Parse arguments
	config := flag.String("config", "", "Config to be used (device value would be ignored if specified)")
	device := flag.String("device", "", "Test device PCI address")
	cniPath := flag.String("cni", defPath, "Provides path to CNI executable")
	testNum := flag.Int("tests", defTestNum, "Number of tests to conduct")
	logFile := flag.String("out", os.Args[0]+".log", "Log file path for successful attempts")
	panicOnly := flag.Bool("panicOnly", defPanicOnly, "Log only Go panics")
	flag.Parse()

	if *device == "" && *config == "" {
		fmt.Println("error: device has to be specified or config file has to be provided")
		os.Exit(1)
	}

	var err error

	// Open logfiles
	f, err := os.Create(*logFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	defer f.Close()

	// Create test namespace
	targetNS, err := testutils.NewNS()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}

	var conf string
	if *config == "" {
		conf = fmt.Sprintf(`{
		"cniVersion": "0.3.0",
		"deviceID":"%s",
		"name": "sriov-net-test",
		"spoofchk":"off",
		"type": "sriov"
		}`, *device)
	} else {
		confBytes, err := ioutil.ReadFile(*config)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(4)
		}
		conf = string(confBytes)
	}

	var sCalls []cmdStatus // successful calls
	var fCalls []cmdStatus // failed calls

	// Perform test
	for i := 0; i < *testNum; i++ {
		// Malform StdinData using radamsa
		radamsaCmd := exec.Command("radamsa")
		radamsaCmd.Stdin = strings.NewReader(conf)
		malformed, err := radamsaCmd.CombinedOutput()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(5)
		}

		// Create args
		args := &skel.CmdArgs{
			ContainerID: "dummy",
			Netns:       targetNS.Path(),
			IfName:      "net1",
			Path:        filepath.Dir(*cniPath),
			StdinData:   malformed,
		}

		// Run CNI with ADD command
		r, err := callCni(*cniPath, "ADD", args, f, *panicOnly)
		if err == nil {
			// If ADD was successfull try to DEL.
			_, err = callCni(*cniPath, "DEL", args, f, *panicOnly)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(6)
			}
		}
		if r.Successful {
			sCalls = append(sCalls, r)
		} else {
			fCalls = append(fCalls, r)
		}
	}

	// Unmount test namespace
	targetNS.Close()
	if err = testutils.UnmountNS(targetNS); err != nil {
		fmt.Println(err.Error())
		os.Exit(7)
	}

	printSummary(&sCalls, &fCalls, *logFile)

	os.Exit(0)
}
