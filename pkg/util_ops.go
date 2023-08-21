package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
)

func UtilOpsProfileList(inipath string, showProfileData bool) {
	winSCPProfiles, err := WinSCPiniExtractProfileData(inipath, "ALL", false)
	if err != nil {
		fmt.Println(err)
	}

	for _, winSCPProfile := range winSCPProfiles {
		if showProfileData {
			fmt.Printf("%s - (%s:%s:%s)\n", winSCPProfile.ProfileName, winSCPProfile.Username, winSCPProfile.Hostname, winSCPProfile.Port)
		} else {
			fmt.Println(winSCPProfile.ProfileName)
		}
	}

}

func UtilsOpsExtractWinSCPProfile(inipath, winscpProfileName string) error {
	winSCPProfiles, err := WinSCPiniExtractProfileData(inipath, winscpProfileName, false)
	if err != nil {
		return err
	}
	jsonByte, err := json.Marshal(&winSCPProfiles)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(jsonByte))

	return nil
}

func UtilsOpsSFTPTestConnection(inipath, winSCPProfileName, DisplayType string, CheckThreads int, ShowThreadsAtWork bool) error {

	// show something while the util is busy in the background
	waitSpinner := spinner.New(spinner.CharSets[9], 100*time.Millisecond)

	if DisplayType == "TABLE" {
		log.Println("Begin testing.")
		if !ShowThreadsAtWork {
			waitSpinner.Start()
		}
	}

	// run actual test execution engine, it's multithreaded by default
	collectedConnTestResults, err := executeConnectionTest(inipath, winSCPProfileName, CheckThreads, ShowThreadsAtWork)
	if err != nil {
		return err
	}

	//
	// reorder the listing by profilefile
	//
	// start with two result lists, success and everything else
	var rsltSuccess, rsltFailure []ConnectionTestResult
	// split the main list into two
	for _, v := range collectedConnTestResults {
		if v.ConnTestResult == "SUCCESS" {
			rsltSuccess = append(rsltSuccess, v)
		} else {
			rsltFailure = append(rsltFailure, v)
		}
	}
	// sort each by the profilefile
	sort.Slice(rsltSuccess, func(i, j int) bool {
		return rsltSuccess[i].ProfileName < rsltSuccess[j].ProfileName
	})
	sort.Slice(rsltFailure, func(i, j int) bool {
		return rsltFailure[i].ProfileName < rsltFailure[j].ProfileName
	})
	// reset and combined.
	collectedConnTestResults = rsltSuccess
	collectedConnTestResults = append(collectedConnTestResults, rsltFailure...)

	//
	// display type
	//
	switch DisplayType {
	case "TABLE":
		log.Println("Completed testing.")

		var table *tablewriter.Table
		// results
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"profilename", "hostname", "port", "username", "auth method", "result", "further info"})
		for _, v := range collectedConnTestResults {
			// fmt.Printf("%v : %v : (%v)", v.ProfileName, v.ConnTestResult, v.FurtherInfo)
			table.Append([]string{v.ProfileName, v.Hostname, v.Port, v.Username, v.AuthMethod, v.ConnTestResult, v.FurtherInfo})
		}
		table.Render()
	case "JSON":

		jsonByte, err := json.Marshal(&collectedConnTestResults)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(jsonByte))
	}

	if waitSpinner.Active() {
		waitSpinner.Stop()
	}

	return nil
}

func SFTPGetFileList(WinSCPINIPath, WinSCPProfileName, SftpRemotePath, SftpSrcFileMask string) error {

	sftpconncfg, err := WinSCPiniExtractProfileData(WinSCPINIPath, WinSCPProfileName, false)
	if err != nil {
		return err
	}

	if len(SftpRemotePath) == 0 {
		if len(sftpconncfg[0].RemotePath) != 0 {
			SftpRemotePath = sftpconncfg[0].RemotePath
		} else {
			SftpRemotePath = "/"
		}
	}

	// check if the remote path ends with a "/"
	if !strings.HasSuffix(SftpRemotePath, "/") {
		SftpRemotePath = SftpRemotePath + "/"
	}

	sftpConn, err := getSFTPConnection(sftpconncfg[0])
	if err != nil {
		return err
	}
	defer sftpConn.Close()

	// remoteFileInfo, err := sftpRemoteList(sftpConn, "GET", SftpRemotePath, SftpSrcFileMask, false, false)
	// if err != nil {
	// 	return err
	// }

	remoteFileInfo, err := sFTPRemoteFileSearchREADDIR(sftpConn, SftpRemotePath, SftpSrcFileMask)
	if err != nil {
		return err
	}

	// type FileInfo interface {
	// 	Name() string       // base name of the file
	// 	Size() int64        // length in bytes for regular files; system-dependent for others
	// 	Mode() FileMode     // file mode bits
	// 	ModTime() time.Time // modification time
	// 	IsDir() bool        // abbreviation for Mode().IsDir()
	// 	Sys() any           // underlying data source (can return nil)
	// }

	log.Println("Preparing infomation table...")

	// var table *tablewriter.Table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"object", "isDir?", "size", "modtime"})
	for _, v := range remoteFileInfo {
		// fmt.Printf("%v : %v : (%v)", v.ProfileName, v.ConnTestResult, v.FurtherInfo)
		table.Append([]string{v.Name(), strconv.FormatBool(v.IsDir()), strconv.FormatInt(v.Size(), 10), v.ModTime().String()})
	}
	table.Render()

	return nil
}

// Separate the "work engine" from the display stuff on the main call
// also gives a useful little raw entry point if needed!
func executeConnectionTest(inipath, winSCPProfileName string, CheckThreads int, ShowThreadsAtWork bool) ([]ConnectionTestResult, error) {

	var collectedConnTestResults []ConnectionTestResult

	if len(winSCPProfileName) == 0 {
		// if none is specified then just get the first one you can find!
		winSCPProfilesTmpList, err := WinSCPiniExtractProfileData(inipath, "ALL", false)
		if err != nil {
			return nil, err
		}

		rndSrc := rand.NewSource(time.Now().UnixNano())
		rndGen := rand.New(rndSrc)

		winSCPProfileName = winSCPProfilesTmpList[rndGen.Intn(len(winSCPProfilesTmpList)-1)].ProfileName
		log.Printf("Using random profile [%s]\n", winSCPProfileName)
	}

	if winSCPProfileName == "ALL" {

		winSCPProfiles, err := WinSCPiniExtractProfileData(inipath, "ALL", false)
		if err != nil {
			return nil, err
		}

		// channels to push profiles to be checked
		//             pull back results after checking
		jobs := make(chan string, len(winSCPProfiles))
		results := make(chan ConnectionTestResult, len(winSCPProfiles))

		//
		//  STAGE 1 - THE SETUP
		//
		// This sets up X number of "worker threads"
		//   they will sit waiting on the "jobs" channel waiting for stuff to come in
		//
		for w := 1; w <= CheckThreads; w++ {

			// you must see channels from "inside the function", jobs are pulled in and results go out
			go func(id int, WinSCPINIPath string, showWork bool, jobs <-chan string, results chan<- ConnectionTestResult) {
				for sftpProfileName := range jobs {
					extractedsftpconn, _ := WinSCPiniExtractProfileData(WinSCPINIPath, sftpProfileName, false)
					testResult := TestSFTPConnection(extractedsftpconn[0])
					if showWork {
						log.Printf("[%d] : [%s] - [%s] : [%s]", id, sftpProfileName, testResult.ConnTestResult, testResult.FurtherInfo)
					}
					results <- testResult
				}
			}(w, inipath, ShowThreadsAtWork, jobs, results)

		}

		//
		//  STAGE 2 - THE PUSH IN
		//
		// now start loading up the profile names into the job queue
		//   this will wake the "worker" threads in the background.
		//   we are NOT executing anythingm here, simply loading up a work queue with things to do
		//   as each worker finishes their current task they go get another job off the queue
		//   once the jobs queue us exhausted, the job threads will stop on their own
		for _, sftpProfileName := range winSCPProfiles {
			jobs <- sftpProfileName.ProfileName
		}
		close(jobs)

		//
		//  STAGE 3 - THE PULL OUT
		//
		// at this point there may or may not be items on the results queue
		// this will keep going until there is nothing left in the results queue
		// we just keep pulling items and putting the results into a slice
		for a := 1; a <= len(winSCPProfiles); a++ {
			collectedConnTestResults = append(collectedConnTestResults, <-results)
		}

	} else {
		extractedsftpconn, err := WinSCPiniExtractProfileData(inipath, winSCPProfileName, false)
		if err != nil {
			return nil, err
		}
		collectedConnTestResults = append(collectedConnTestResults, TestSFTPConnection(extractedsftpconn[0]))
	}

	return collectedConnTestResults, nil
}
