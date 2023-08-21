/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// testconnCmd represents the testconn command
var testconnCmd = &cobra.Command{
	Use:   "testconnection",
	Short: "Test profile connection.",
	Long:  `Long desc TBC`,
	Run: func(cmd *cobra.Command, args []string) {

		inipath, err := cmd.Flags().GetString("inipath")
		if err != nil {
			log.Println(" Invalid INI file path. Aborting.")
		}

		winscpProfileName, err := cmd.Flags().GetString("profilename")
		if err != nil {
			log.Printf(" Invalid default profile [%s]. Aborting.", winscpProfileName)
		}

		displayType, err := cmd.Flags().GetString("displaytype")
		if err != nil {
			log.Println(" Invalid output type. Aborting.")
		}

		checkThreads, err := cmd.Flags().GetInt("checkerthreads")
		if err != nil {
			log.Println(" Invalid thread setting. Aborting.")
		}

		showThreadsAtWork, err := cmd.Flags().GetBool("showthreads")
		if err != nil {
			log.Println(" Invalid show working threads option. Aborting.")
		}

		err = pkg.UtilsOpsSFTPTestConnection(inipath, winscpProfileName, strings.ToUpper(displayType), checkThreads, showThreadsAtWork)
		if err != nil {
			log.Println(err)
		}

	},
}

func init() {
	rootCmd.AddCommand(testconnCmd)
	testconnCmd.Flags().StringP("profilename", "p", "", "Name of WinSCP profile to test ( use profile option for list or use \"ALL\" ) (def: random)")
	testconnCmd.Flags().StringP("displaytype", "d", "TABLE", "Type of display output TABLE or JSON")
	testconnCmd.Flags().IntP("checkerthreads", "t", 3, "Number of concurrent checker threads active at once")
	testconnCmd.Flags().BoolP("showthreads", "s", false, "Show output from threads while they work. ( def : off )")

}
