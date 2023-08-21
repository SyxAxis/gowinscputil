/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extractprofileasjson",
	Short: "Extract WinSCP profile as JSON",
	Long: `The extract command locates and extracts all the required details 
from WinSCP.ini files that are required to make a connection to the remote SFTP site.`,
	Run: func(cmd *cobra.Command, args []string) {

		inipath, _ := cmd.Flags().GetString("inipath")
		winscpProfileName, _ := cmd.Flags().GetString("profilename")

		pkg.UtilsOpsExtractWinSCPProfile(inipath, winscpProfileName)

	},
}

func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmd.Flags().StringP("profilename", "p", "ALL", "Name of WinSCP profile to extract or use ALL(def)")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// extractCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// extractCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	//.BoolP("toggle", "t", false, "Help message for toggle")
}
