/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// listprofilesCmd represents the listprofiles command
var listprofilesCmd = &cobra.Command{
	Use:   "getprofilelist",
	Short: "List the profile names in a WinSCP.ini file.",
	Long:  `Long desc TBC`,
	Run: func(cmd *cobra.Command, args []string) {

		inipath, _ := cmd.Flags().GetString("inipath")
		showProfileMetaData, _ := cmd.Flags().GetBool("showmetadata")

		pkg.UtilOpsProfileList(inipath, showProfileMetaData)
	},
}

func init() {
	rootCmd.AddCommand(listprofilesCmd)

	listprofilesCmd.Flags().BoolP("showmetadata", "s", false, "Show additional information about the profile ( def: false ).")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listprofilesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listprofilesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
