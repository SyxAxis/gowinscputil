/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// getlistCmd represents the getlist command
var getlistCmd = &cobra.Command{
	Use:   "getremotefilelist",
	Short: "Connect to the server, fetch list of files and folders",
	Long:  `Long desc.`,
	Run: func(cmd *cobra.Command, args []string) {

		inipath, err := cmd.Flags().GetString("inipath")
		if err != nil {
			log.Println(" Invalid INI file path. Aborting.")
		}

		winscpProfileName, err := cmd.Flags().GetString("profilename")
		if err != nil {
			log.Printf(" Invalid default profile [%s]. Aborting.", winscpProfileName)
		}

		sftpremotepath, err := cmd.Flags().GetString("sftpremotepath")
		if err != nil {
			log.Printf(" Invalid default remote path [%s]. Aborting.", sftpremotepath)
		}

		sftpsrfilemask, err := cmd.Flags().GetString("sftpsrfilemask")
		if err != nil {
			log.Printf(" Invalid default file mask [%s]. Aborting.", sftpsrfilemask)
		}

		err = pkg.SFTPGetFileList(inipath, winscpProfileName, sftpremotepath, sftpsrfilemask)
		if err != nil {
			log.Println(err)
		}

	},
}

func init() {
	rootCmd.AddCommand(getlistCmd)
	getlistCmd.Flags().StringP("profilename", "p", "NONE", "Name of WinSCP profile to test ( use listprofiles for list or ALL )")
	getlistCmd.Flags().StringP("sftpremotepath", "r", "", "Remote SFTP path ( def \"/\" )")
	getlistCmd.Flags().StringP("sftpsrfilemask", "m", "^*", "Remote SFTP path as a regex")

}
