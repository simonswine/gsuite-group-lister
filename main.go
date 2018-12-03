package main

import (
	"context"
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/simonswine/gsuite-group-lister/google"
)

var googleServiceAccountPath string
var googleImpersonateAdmin string

var rootCmd = &cobra.Command{
	Use:   "gsuite-group-lister",
	Short: "Lists gsuite groups and memberships",
	Run: func(cmd *cobra.Command, args []string) {
		gp, err := google.New(googleServiceAccountPath)
		if err != nil {
			panic(err)
		}
		gp.ImpersonateAdmin = googleImpersonateAdmin

		groups, err := gp.ListGroups(context.Background())
		if err != nil {
			panic(err)
		}

		for _, group := range groups {
			fmt.Printf("%s%v\n", group.String(), group.Members)
		}
		// Do Stuff Here
	},
}

func init() {
	googleServiceAccountPathDefault, err := homedir.Expand("~/.config/gcloud/terraform-admin.json")
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().StringVarP(&googleServiceAccountPath, "google-service-account-path", "s", googleServiceAccountPathDefault, "path to the google service account file")
	rootCmd.PersistentFlags().StringVarP(&googleImpersonateAdmin, "google-impersonate-admin", "a", "christian.simon@jetstack.io", "admin user to impersonate when using the directory API")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
