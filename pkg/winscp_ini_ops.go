package pkg

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var config *viper.Viper

const (
	PW_MAGIC = 0xA3
	PW_FLAG  = 0xFF
)

func WinSCPiniExtractProfileData(inipath string, winSCPProfileName string, maskSensitiveData bool) ([]WinSCPProfile, error) {
	config, err := initViperConfigReader(config, inipath)
	if err != nil {
		fmt.Println(err)
	}

	// dump the important key data back to a slice of structs
	winSCPProfiles, err := extractProfile(config, winSCPProfileName, maskSensitiveData)
	if err != nil {
		return nil, err
	}
	return winSCPProfiles, nil
}

func initViperConfigReader(conf *viper.Viper, filePath string) (*viper.Viper, error) {

	conf = viper.New()

	conf.SetConfigType("ini")
	conf.AddConfigPath("./")
	conf.SetConfigFile(filePath)
	// fmt.Printf("Using config: %s\n", conf.ConfigFileUsed())

	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "\\"))

	err := conf.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}

	// conf.WatchConfig()
	// conf.OnConfigChange(func(e fsnotify.Event) {
	// 	fmt.Printf("Config file changed: %v\n", e.Name)
	// })

	return conf, nil
}

func decryptWinSCPProfilePassword(host, username, password string) string {
	key := username + host
	passbytes := []byte{}
	for i := 0; i < len(password); i++ {
		val, _ := strconv.ParseInt(string(password[i]), 16, 8)
		passbytes = append(passbytes, byte(val))
	}
	var flag byte
	flag, passbytes = decryptWinSCPProfilePasswordNextChar(passbytes)
	var length byte = 0
	if flag == PW_FLAG {
		_, passbytes = decryptWinSCPProfilePasswordNextChar(passbytes)

		length, passbytes = decryptWinSCPProfilePasswordNextChar(passbytes)
	} else {
		length = flag
	}

	toBeDeleted, passbytes := decryptWinSCPProfilePasswordNextChar(passbytes)
	passbytes = passbytes[toBeDeleted*2:]

	clearpass := ""
	var (
		i   byte
		val byte
	)
	for i = 0; i < length; i++ {
		val, passbytes = decryptWinSCPProfilePasswordNextChar(passbytes)
		clearpass += string(val)
	}

	if flag == PW_FLAG {
		clearpass = clearpass[len(key):]
	}
	return clearpass
}

func decryptWinSCPProfilePasswordNextChar(passbytes []byte) (byte, []byte) {
	if len(passbytes) <= 0 {
		return 0, passbytes
	}
	a := passbytes[0]
	b := passbytes[1]
	passbytes = passbytes[2:]
	return ^(((a << 4) + b) ^ PW_MAGIC) & 0xff, passbytes
}

func extractProfile(config *viper.Viper, profilenamereq string, maskSensitiveData bool) ([]WinSCPProfile, error) {

	// =====================================================================
	// very long winded but sadly the INI is very raw
	// so we sift out a unique list of profiles that makes it
	// way easier to pull out the actual data

	// using a map to make use of keys uniqueness to remove dupes
	// temp just to make it easier to sift later
	profiles := make(map[string]bool)

	for _, x := range config.AllKeys() {

		// for ALL just keep going
		if strings.ToUpper(profilenamereq) == "ALL" {
			if strings.Contains(x, ".hostname") && !strings.Contains(x, "to_be_decom") {
				profiles[strings.ReplaceAll(x, ".hostname", "")] = true
			}
		}

		// single entry, break if/when you find it
		if strings.Compare(x, "sessions\\"+strings.ToLower(profilenamereq)+".hostname") == 0 {
			profiles[strings.ReplaceAll(x, ".hostname", "")] = true
			break
		}

	}

	// failed to find anything!
	if len(profiles) == 0 {
		return nil, errors.New(fmt.Sprintf(" Unable to locate requested profile [%s].", profilenamereq))
	}

	// now build a string slice from the map
	keys := make([]string, 0, len(profiles))
	for k := range profiles {
		keys = append(keys, k)
	}
	// sort the string slice into order
	sort.Strings(keys)

	// =====================================================================
	//

	// =====================================================================
	// now we have a unique and sorted list of profile names ( "stubs") we can cycle

	var WinSCPProfilesSet []WinSCPProfile

	for _, v := range keys {

		var hostname, username, clear_text_pass, portnumber, protocol, proxyhost, proxyport, ppkfilepath, ppkrawdata, hostfingerprint, remotepath, localpath string

		hostname = config.GetString(v + ".hostname")
		username = config.GetString(v + ".username")

		if config.IsSet(v + ".password") {
			if maskSensitiveData {
				clear_text_pass = "********"
			} else {
				clear_text_pass = decryptWinSCPProfilePassword(config.GetString(v+".hostname"), config.GetString(v+".username"), config.GetString(v+".password"))
			}
		} else {
			clear_text_pass = ""
		}

		if config.IsSet(v + ".fsprotocol") {
			if config.GetString(v+".fsprotocol") == "2" {
				protocol = "SFTP"
			} else if config.GetString(v+".fsprotocol") == "5" {
				protocol = "FTP"
			} else {
				protocol = "UNKNOWN"
			}
		} else {
			// if not set then default is SFTP
			protocol = "SFTP"
		}

		if config.IsSet(v + ".portnumber") {
			portnumber = config.GetString(v + ".portnumber")
		} else {
			portnumber = "22"
		}

		if config.IsSet(v + ".proxyhost") {
			proxyhost = config.GetString(v + ".proxyhost")
		}

		if config.IsSet(v + ".proxyport") {
			proxyport = config.GetString(v + ".proxyport")
		}

		if config.IsSet(v + ".publickeyfile") {

			if maskSensitiveData {
				ppkfilepath = "********"
				ppkrawdata = "********"
			} else {

				ppkfilepath = strings.ReplaceAll(config.GetString(v+".publickeyfile"), "%20", " ")
				ppkfilepath = strings.ReplaceAll(ppkfilepath, "%5C", "\\")

				// try get the putty key data and stash it as a string
				if _, err := os.Stat(ppkfilepath); err == nil {
					ppkfilerawdata, err := ConvertPuttyFormattedKey(ppkfilepath, false)
					if err != nil {
						ppkrawdata = fmt.Sprintf("*** Error reading key file : %s ***", ppkfilepath)
					} else {
						ppkrawdata = string(ppkfilerawdata)
					}
				} else if errors.Is(err, os.ErrNotExist) {
					ppkrawdata = fmt.Sprintf("*** Cannot locate key file : %s ***", ppkfilepath)
				}

			}

		} else {
			ppkfilepath = ""
		}

		if config.IsSet("configuration\\lastfingerprints." + config.GetString(v+".hostname")) {
			if maskSensitiveData {
				hostfingerprint = "********"
			} else {
				hostfpref := config.GetString("configuration\\lastfingerprints." + config.GetString(v+".hostname"))
				hostfingerprint = strings.ReplaceAll(strings.Split(hostfpref, "=")[1], "%20", " ") + "="
			}
		} else {
			hostfingerprint = ""
		}

		if config.IsSet(v + ".RemoteDirectory") {
			localpath = strings.ReplaceAll(config.GetString(v+".RemoteDirectory"), "%20", " ")
			remotepath = config.GetString(v + ".RemoteDirectory")
			// add as trailing slash if needed
			if !strings.HasSuffix(remotepath, "/") {
				remotepath = remotepath + "/"
			}
		} else {
			remotepath = "/"
		}

		if config.IsSet(v + ".LocalDirectory") {
			localpath = strings.ReplaceAll(config.GetString(v+".LocalDirectory"), "%20", " ")
			localpath = strings.ReplaceAll(localpath, "%5C", "\\")
			// add as trailing slash if needed
			if !strings.HasSuffix(localpath, "\\") {
				localpath = localpath + "\\"
			}

		} else {
			localpath = "C:\\"
		}

		// fmt.Println("===================================================================")
		// hostpublickeys are overridden by the prefix for the public key hash type
		// Viper is using ":" to delimit when ti should be "="
		// this means viper.sshhostkeys only has around 7 unique keys instead of 100 expected
		// fmt.Println("===================================================================")

		// fmt.Println("===================================================================")
		// // for x, y := range config.AllKeys() {
		// // 	"sshhostkeys.ssh-ed25519@22"
		// // 	fmt.Println("CFGVAL X : ", x)
		// // 	fmt.Println("CFGVAL Y : ", y)
		// // }
		// // for cfgkey, cfgval := range config.AllSettings() {

		// // 	if cfgkey == "sshhostkeys" {
		// // 		for hostkeytype, hostkeyval := range cfgval.(map[string]interface{}) {
		// // 			fmt.Println("CFGVAL X : ", hostkeytype)
		// // 			fmt.Println("CFGVAL Y : ", hostkeyval)
		// // 		}
		// // 	}

		// // 	// match, _ := regexp.MatchString("SshHostKeys.[a-z0-9]*@"+portnumber+":"+hostname, cfgkey)
		// // 	// if match {
		// // 	// 	hostkeyref = config.GetString(cfgkey)
		// // 	// }
		// // }
		// fmt.Println("===================================================================")

		// match, _ := regexp.MatchString("SshHostKeys.[a-z0-9]*@"+portnumber+":"+hostname, v)
		// if match {
		// 	hostfpref := config.GetString("configuration\\lastfingerprints." + config.GetString(v+".hostname"))
		// 	hostfingerprint = strings.ReplaceAll(strings.Split(hostfpref, "=")[1], "%20", " ")
		// } else {
		// 	hostfingerprint = ""
		// }

		WinSCPProfilesSet = append(WinSCPProfilesSet, WinSCPProfile{
			ProfileName:       strings.Split(v, "\\")[1],
			Hostname:          hostname,
			Username:          username,
			Password:          clear_text_pass,
			Port:              portnumber,
			Protocol:          protocol,
			ProxyServer:       proxyhost,
			ProxyPort:         proxyport,
			PublicKeyFilePath: ppkfilepath,
			PublicKeyRAWText:  ppkrawdata,
			// HostPublicKey:     hostkeyref,
			HostFingerPrint: hostfingerprint,
			RemotePath:      remotepath,
			LocalPath:       localpath,
		})

	}
	// =====================================================================

	return WinSCPProfilesSet, nil
}
