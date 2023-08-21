/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// convkeyCmd represents the convkey command
var convkeyCmd = &cobra.Command{
	Use:   "convertputtykey2openssh",
	Short: "Convert Putty private key file to OpenSSH format",
	Long:  `Long desc TBC`,
	Run: func(cmd *cobra.Command, args []string) {

		puttyKeyFile, _ := cmd.Flags().GetString("puttykeypath")
		showkey, _ := cmd.Flags().GetBool("showkey")

		if showkey {
			pkg.ConvertPuttyFormattedKey(puttyKeyFile, showkey)
		}

	},
}

func init() {
	rootCmd.AddCommand(convkeyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convkeyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	convkeyCmd.Flags().StringP("keypath", "k", "", "Path (inc file) to Putty formatted private key file.")
	convkeyCmd.Flags().BoolP("showkey", "s", true, "Show the converted key")
}
