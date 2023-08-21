package pkg

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

/*

	Designed to be set of standard SFTP functions such as
	connection check, file lists and simple file copy

	12-JAN-2023 - Bit of a mess right now!

*/

func TestSFTPConnection(sftpconncfg WinSCPProfile) ConnectionTestResult {

	var tmpConnTestRslt ConnectionTestResult

	tmpConnTestRslt.ProfileName = sftpconncfg.ProfileName
	tmpConnTestRslt.Hostname = sftpconncfg.Hostname
	tmpConnTestRslt.Username = sftpconncfg.Username
	tmpConnTestRslt.Port = sftpconncfg.Port
	tmpConnTestRslt.AuthMethod = "Password"
	if len(sftpconncfg.PublicKeyFilePath) > 0 {
		tmpConnTestRslt.AuthMethod = "Private key file"
	}

	//
	// FTP connection is tested differently
	//
	if sftpconncfg.Protocol != "SFTP" {
		return testInsecureFTPConnection(sftpconncfg)
		// log.Printf("Connection   : [%v]\n", sftpconncfg.Username+"@"+sftpconncfg.Hostname+":"+sftpconncfg.Port)
		// log.Println("Information  : Non SFTP protocol")
		// tmpConnTestRslt.ConnTestResult = "SKIPPED"
		// tmpConnTestRslt.FurtherInfo = "Non SFTP protocol"
		// return tmpConnTestRslt
	}

	sftpConn, err := getSFTPConnection(sftpconncfg)
	if err != nil {
		tmpConnTestRslt.ConnTestResult = "FAILURE"
		tmpConnTestRslt.FurtherInfo = err.Error()
		return tmpConnTestRslt
	}
	defer sftpConn.Close()

	currdir, err := sftpConn.Getwd()
	if err == nil {
		tmpConnTestRslt.ConnTestResult = "SUCCESS"
		// tmpConnTestRslt.FurtherInfo = ""
		dirlist, err := sftpConn.ReadDir(currdir)
		if err == nil {
			tmpConnTestRslt.FurtherInfo = fmt.Sprintf("Found [%s] file objects.", strconv.Itoa(len(dirlist)))
		} else {
			tmpConnTestRslt.FurtherInfo = "Warning: Unable to obtain file list."
		}
		return tmpConnTestRslt
	}

	_, err = sftpConn.Getwd()
	if err == nil {
		tmpConnTestRslt.ConnTestResult = "SUCCESS"
		tmpConnTestRslt.FurtherInfo = ""
		return tmpConnTestRslt
	}

	tmpConnTestRslt.ConnTestResult = "FAILURE"
	tmpConnTestRslt.FurtherInfo = "Unknown"
	return tmpConnTestRslt

}

func testInsecureFTPConnection(sftpconncfg WinSCPProfile) ConnectionTestResult {

	var tmpConnTestRslt ConnectionTestResult
	var activeServer, activePort, activeUserAcct, activePassword string

	if len(sftpconncfg.ProxyServer) > 0 {
		activeServer = sftpconncfg.ProxyServer
		activePort = sftpconncfg.ProxyPort
		activeUserAcct = sftpconncfg.Username + "@" + sftpconncfg.Hostname
		activePassword = sftpconncfg.Password
		tmpConnTestRslt.Port = sftpconncfg.ProxyPort
	} else {
		activeServer = sftpconncfg.Hostname
		activePort = sftpconncfg.Port
		activeUserAcct = sftpconncfg.Username
		activePassword = sftpconncfg.Password
	}

	tmpConnTestRslt.ProfileName = sftpconncfg.ProfileName
	tmpConnTestRslt.Hostname = activeServer
	tmpConnTestRslt.Username = activeUserAcct
	tmpConnTestRslt.Port = activePort
	tmpConnTestRslt.AuthMethod = "Password"

	c, err := ftp.Dial(activeServer+":"+activePort, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		tmpConnTestRslt.ConnTestResult = "FAILURE"
		tmpConnTestRslt.FurtherInfo = err.Error()
		return tmpConnTestRslt
	}
	err = c.Login(activeUserAcct, activePassword)
	if err != nil {
		tmpConnTestRslt.ConnTestResult = "FAILURE"
		tmpConnTestRslt.FurtherInfo = err.Error()
		return tmpConnTestRslt
	}

	currdir, err := c.CurrentDir()
	if err == nil {
		tmpConnTestRslt.ConnTestResult = "SUCCESS"
		// tmpConnTestRslt.FurtherInfo = ""
		dirlist, err := c.List(currdir)
		if err == nil {
			tmpConnTestRslt.FurtherInfo = fmt.Sprintf("Found [%s] file objects.", strconv.Itoa(len(dirlist)))
		} else {
			tmpConnTestRslt.FurtherInfo = "Warning: Unable to obtain file list."
		}
		_ = c.Quit()
		return tmpConnTestRslt
	}

	tmpConnTestRslt.ConnTestResult = "FAILURE"
	tmpConnTestRslt.FurtherInfo = "Unknown"
	_ = c.Quit()
	return tmpConnTestRslt

	// // Do something with the FTP conn

	// if err := c.Quit(); err != nil {
	// 	log.Fatal(err)
	// }

}

// ===========================================================================

func getSFTPConnection(activeSFTPConnConfig WinSCPProfile) (*sftp.Client, error) {

	var sftpAuthConfig *ssh.ClientConfig
	// var keyErr *knownhosts.KeyError

	// pull in the private key file data
	// If using PuttyGen to make keys, export private key as OpenSSH format
	if len(activeSFTPConnConfig.PublicKeyFilePath) > 0 {
		// log.Println("Auth method  : Private key file.")
		// log.Printf("Keyfile : %v", activeSFTPConnConfig.PublicKeyFilePath)

		// WinSCP only works with Putty formatted private keys,
		//    they need to be reformatted into OpenSSH

		// privPEMBytes := ConvertPuttyFormattedKey("test_RSA2048_priv_PUTTY.priv", false)
		privPEMBytes, err := ConvertPuttyFormattedKey(activeSFTPConnConfig.PublicKeyFilePath, false)
		if err != nil {
			return nil, err
		}

		// parse the private key to make sure it's sound
		signer, err := ssh.ParsePrivateKey(privPEMBytes)
		if err != nil {
			return nil, err
		}

		// ==================================================================================
		//      WARNING -- WARNING -- WARNING -- WARNING -- WARNING -- WARNING -- WARNING
		// ==================================================================================
		//
		// This uses "InsecureIgnoreHostKey" this should not be used except for testing
		//   it allows the remote site to simply be accept as-is. The WinSCP.ini has
		//   the host keys and fingerprints, they just need to be extracted and used somehow
		//
		// I have the code to keep a standardise known_hosts file upto date but need to test
		//  the hostkey callback function
		//
		// ==================================================================================

		// attach using a private key and a valid hostkey string
		// Note: INSECURE connection ignoring the remote host's hostkey
		//       will give your sec admin a fit!!
		sftpAuthConfig = &ssh.ClientConfig{
			User: activeSFTPConnConfig.Username,
			// alterntaive is to use a plain text password
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
				ssh.Password(activeSFTPConnConfig.Password),
			},
			// ignore the hostkey and just accept it, NOT a good idea in prod/live envs
			// especially on the nasty internet!
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Second * 10,
		}

		// // attach using a private key and a valid hostkey string
		// // Note: INSECURE connection ignoring the remote host's hostkey
		// //       will give your sec admin a fit!!
		// sftpAuthConfig = &ssh.ClientConfig{
		// 	User: activeSFTPConnConfig.Username,
		// 	// alterntaive is to use a plain text password
		// 	Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// 	// ignore the hostkey and just accept it, NOT a good idea in prod/live envs
		// 	// especially on the nasty internet!
		// 	// hostkey "pubkey" is a BIG int decimal expression of the keys hex val
		// 	// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// 	// HostKeyCallback: ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
		// 	// 	fmt.Println("CB H: ", host)
		// 	// 	fmt.Println("CB R: ", remote)
		// 	// 	fmt.Println("CB P: ", pubKey)
		// 	// 	return nil
		// 	// }),
		// 	HostKeyCallback: ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
		// 		kh := checkKnownHosts()
		// 		hErr := kh(host, remote, pubKey)
		// 		// Reference: https://blog.golang.org/go1.13-errors
		// 		// To understand what errors.As is.
		// 		if errors.As(hErr, &keyErr) && len(keyErr.Want) > 0 {
		// 			// Reference: https://www.godoc.org/golang.org/x/crypto/ssh/knownhosts#KeyError
		// 			// if keyErr.Want slice is empty then host is unknown, if keyErr.Want is not empty
		// 			// and if host is known then there is key mismatch the connection is then rejected.
		// 			log.Printf("WARNING: %v is not a key of %s, either a MiTM attack or %s has reconfigured the host pub key.", pubKey, host, host)
		// 			return keyErr
		// 		} else if errors.As(hErr, &keyErr) && len(keyErr.Want) == 0 {
		// 			// host key not found in known_hosts then give a warning and continue to connect.
		// 			log.Printf("WARNING: %s is not trusted, adding this key: %q to known_hosts file.", host, pubKey)
		// 			return addHostKey(host, remote, pubKey)
		// 		}
		// 		log.Printf("Pub key exists for %s.", host)
		// 		return nil
		// 	}),
		// }

	} else if len(activeSFTPConnConfig.Password) > 0 {
		// log.Println("Auth method  : Account password")
		// Note: INSECURE connection ignoring the remote host's hostkey
		//       will give your sec admin a fit!!
		sftpAuthConfig = &ssh.ClientConfig{
			User: activeSFTPConnConfig.Username,
			// alterntaive is to use a plain text password
			Auth: []ssh.AuthMethod{ssh.Password(activeSFTPConnConfig.Password)},
			// ignore the hostkey and just accept it, NOT a good idea in prod/live envs
			// especially on the nasty internet!
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Second * 10,
			// HostKeyCallback: ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
			// 	kh := checkKnownHosts()
			// 	hErr := kh(host, remote, pubKey)
			// 	// Reference: https://blog.golang.org/go1.13-errors
			// 	// To understand what errors.As is.
			// 	if errors.As(hErr, &keyErr) && len(keyErr.Want) > 0 {
			// 		// Reference: https://www.godoc.org/golang.org/x/crypto/ssh/knownhosts#KeyError
			// 		// if keyErr.Want slice is empty then host is unknown, if keyErr.Want is not empty
			// 		// and if host is known then there is key mismatch the connection is then rejected.
			// 		log.Printf("WARNING: %v is not a key of %s, either a MiTM attack or %s has reconfigured the host pub key.", pubKey, host, host)
			// 		return keyErr
			// 	} else if errors.As(hErr, &keyErr) && len(keyErr.Want) == 0 {
			// 		// host key not found in known_hosts then give a warning and continue to connect.
			// 		log.Printf("WARNING: %s is not trusted, adding this key: %q to known_hosts file.", host, pubKey)
			// 		return addHostKey(host, remote, pubKey)
			// 	}
			// 	log.Printf("Pub key exists for %s.", host)
			// 	return nil
			// }),
		}

	} else {
		log.Println("Unable to find either a valid key file config or password. Exiting.")
		return nil, nil
	}

	// establish the SSH connection. SFTP is really just FTP over an SSH tunnel
	// open the SSH tunnel
	conn, err := ssh.Dial("tcp", activeSFTPConnConfig.Hostname+":"+activeSFTPConnConfig.Port, sftpAuthConfig)
	if err != nil {
		// log.Printf("DIAL ERROR: %v", err)
		return nil, err
	}

	// log.Printf("Connected to : [%v]\n", conn.RemoteAddr().String())

	// once the ssh hooked up, attach the SFTP connection down the SSH tunnel
	client, err := sftp.NewClient(conn)
	if err != nil {
		// log.Printf("  NC ERROR: %v", err)
		return nil, err
	}

	// get the connection back
	return client, nil

}

func sftpRemoteList(sftpConn *sftp.Client, SftpTransferDirection string, SftpRemotePath string, SftpSrcFileMask string, SftpGetLatestSrcFileOnly bool, SftpRaiseSrcFileMissingAsError bool) ([]fs.FileInfo, error) {

	var srcFilesFound []string
	var err error

	var srcFolderPath string

	if strings.ToUpper(SftpTransferDirection) == "GET" {

		srcFolderPath = SftpRemotePath

		log.Printf("Scanning for source file pattern [%s]\n", srcFolderPath+SftpSrcFileMask)
		// get file list
		srcFilesFound, err = sFTPRemoteFileSearchGLOB(sftpConn, srcFolderPath, SftpSrcFileMask, SftpGetLatestSrcFileOnly)
		if err != nil {
			fmt.Println(err)
		}
	}

	if len(srcFilesFound) == 0 && SftpRaiseSrcFileMissingAsError {
		log.Printf("Found [%v] matching remote file/folder obejcts and option flagSrcFileMissingAsError:enabled.\n", len(srcFilesFound))
		os.Exit(1)
	} else {
		log.Printf("Found [%v] remote file/folder objects...\n", len(srcFilesFound))
	}

	var foundFileStatistics []fs.FileInfo

	// var fileNameToTrx string
	// var srcFilePath string
	for _, srcFileFullName := range srcFilesFound {

		rmtFileInfo, err := sftpConn.Stat(srcFolderPath + filepath.Base(srcFileFullName))
		if err != nil {
			return nil, err
		}
		foundFileStatistics = append(foundFileStatistics, rmtFileInfo)

		// fileNameToTrx = filepath.Base(srcFileFullName)
		// srcFilePath = srcFolderPath + fileNameToTrx
		// // change this to a regex
		// // if the targetfile needs to be fixed then make sure there's only one source file

		// // if len(srcFilesFound) == 1 {
		// // } else {
		// // 	log.Fatalf("[%v] matching files at source. There must only be one matching file at source. Aborting.\n", len(srcFilesFound))
		// // }

		// fileInfo, err := sftpConn.Stat(srcFilePath)
		// if err != nil {
		// 	return err
		// } else {
		// 	log.Printf("Remote file : [%v] modtime: [%v] bytes: [%v] \n", srcFilePath, fileInfo.ModTime(), fileInfo.Size())
		// }

	}

	return foundFileStatistics, nil
}

func sFTPRemoteFileSearchGLOB(sftpConn *sftp.Client, srcFolderPath string, srcFileMask string, getLatestSrcFile bool) ([]string, error) {

	srcFilesFound, err := sftpConn.Glob(srcFolderPath + srcFileMask)
	if err != nil {
		return nil, err
	}

	if getLatestSrcFile {
		log.Println("Get latest file option enabled.")
		var latestFileName []string
		var currSrcFile os.FileInfo

		if len(srcFilesFound) > 0 {
			for _, srcFilename := range srcFilesFound {
				// get the metadata for the latest file off the stack
				fileInfo, err := sftpConn.Stat(srcFilename)
				if err != nil {
					fmt.Println(err)
				}

				// on the first run through the currSrcFile has no value
				//  if that's true then don't bother checking it, just accept it
				if currSrcFile != nil {
					// if the file
					if fileInfo.ModTime().After(currSrcFile.ModTime()) {
						currSrcFile = fileInfo
					}
				} else {
					currSrcFile = fileInfo
				}
			}
			latestFileName = append(latestFileName, currSrcFile.Name())
		}
		return latestFileName, nil
	} else {
		return srcFilesFound, nil
	}

}

func sFTPRemoteFileSearchREADDIR(sftpConn *sftp.Client, sftpRemotePath string, srcFileMask string) ([]fs.FileInfo, error) {

	var matchedfiles []fs.FileInfo

	remoteFileInfo, err := sftpConn.ReadDir(sftpRemotePath)
	if err != nil {
		return nil, err
	}

	validRegex := regexp.MustCompile(srcFileMask)

	for _, tmpfileinfo := range remoteFileInfo {
		if validRegex.MatchString(tmpfileinfo.Name()) {
			matchedfiles = append(matchedfiles, tmpfileinfo)
		}
	}

	return matchedfiles, nil

}

func createKnownHosts() {
	f, fErr := os.OpenFile(filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "known_hosts"), os.O_CREATE, 0600)
	if fErr != nil {
		log.Fatal(fErr)
	}
	f.Close()
}

func checkKnownHosts() ssh.HostKeyCallback {
	createKnownHosts()
	kh, e := knownhosts.New(filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "known_hosts"))
	if e != nil {
		log.Fatal(e)
	}
	return kh
}
func addHostKey(host string, remote net.Addr, pubKey ssh.PublicKey) error {
	// add host key if host is not found in known_hosts, error object is return, if nil then connection proceeds,
	// if not nil then connection stops.
	khFilePath := filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "known_hosts")

	f, fErr := os.OpenFile(khFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if fErr != nil {
		return fErr
	}
	defer f.Close()

	knownHosts := knownhosts.Normalize(remote.String())
	_, fileErr := f.WriteString(knownhosts.Line([]string{knownHosts}, pubKey))
	return fileErr
}

// func TestSFTPConnectionByKey() {

// 	var sftpAuthConfig *ssh.ClientConfig

// 	var SFTPhost, SFTPport, SFTPuser string

// 	SFTPhost = "dba-mgmt4"
// 	SFTPport = "22"
// 	SFTPuser = "gxj"

// 	// pull in the private key file data
// 	// If using PuttyGen to make keys, export private key as OpenSSH format
// 	// if len(activeSFTPConnConfig.SFTPprivatekeyfile) > 0 {
// 	log.Println("Using private key file for authentication.")
// 	// var signer ssh.Signer
// 	// var pemBytes []byte

// 	// pemBytes, err := ioutil.ReadFile(SFTPprivatekeyfile)
// 	privPEMBytes := ConvertPuttyFormattedKey("test_RSA2048_priv_PUTTY.priv", false)
// 	// "test_RSA2048_priv_PUTTY.ppk"
// 	// if err != nil {
// 	// 	fmt.Println(err)
// 	// }

// 	// parse the private key to make sure it's sound
// 	signer, err := ssh.ParsePrivateKey(privPEMBytes)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	// attach using a private key and a valid hostkey string
// 	// Note: INSECURE connection ignoring the remote host's hostkey
// 	//       will give your sec admin a fit!!
// 	sftpAuthConfig = &ssh.ClientConfig{
// 		User: SFTPuser,
// 		// alterntaive is to use a plain text password
// 		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
// 		// ignore the hostkey and just accept it, NOT a good idea in prod/live envs
// 		// especially on the nasty internet!
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	// establish the SSH connection. SFTP is really just FTP over an SSH tunnel
// 	// open the SSH tunnel
// 	conn, err := ssh.Dial("tcp", SFTPhost+":"+SFTPport, sftpAuthConfig)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	log.Printf("Connected to : [%v]\n", conn.RemoteAddr().String())

// 	// once the ssh hooked up, attach the SFTP connection down the SSH tunnel
// 	client, err := sftp.NewClient(conn)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	// fmt.Println(client)

// 	// get the connection back
// 	// return client

// 	err = sftpRemoteListOnly(client, "GET", "/home/gxj/", "automic_uat_02T.tgz", true, false)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// }
