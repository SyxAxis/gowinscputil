/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/syxaxis/winscputil/pkg"
)

// restserverCmd represents the restserver command
var restserverCmd = &cobra.Command{
	Use:   "startrestserver",
	Short: "Start a REST server ( with SSL ) that can send the info back over HTTPS",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		inipath, err := cmd.Flags().GetString("inipath")
		if err != nil {
			log.Println(" Invalid INI file path. Aborting.")
		}

		restSvrPort, err := cmd.Flags().GetInt("port")
		if err != nil {
			log.Println(" Invalid REST server port. Aborting.")
		}

		restSSL, err := cmd.Flags().GetBool("ssl")
		if err != nil {
			log.Println(" Invalid SSL selection. Aborting.")
		}

		pkg.InitRESTServer(inipath, restSvrPort, restSSL)
	},
}

func init() {
	rootCmd.AddCommand(restserverCmd)

	restserverCmd.Flags().Int("port", 8080, "Port on which to run REST server.")
	restserverCmd.Flags().Bool("ssl", true, "Disable TLS/SSL. ( Expects restserver.crt and restserver.key )")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// restserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// restserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
