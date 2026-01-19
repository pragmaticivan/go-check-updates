package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/app"
	"github.com/pragmaticivan/go-check-updates/internal/scanner"
	"github.com/pragmaticivan/go-check-updates/internal/tui"
	"github.com/pragmaticivan/go-check-updates/internal/updater"
	"github.com/spf13/cobra"
)

var (
	// Flags
	upgradeFlag         bool
	verifyFlag          bool // Interactive mode (verify/select); using -i
	filterFlag          string
	allFlag             bool
	cooldownFlag        int
	formatFlag          string
	vulnerabilitiesFlag bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gcu",
	Short: "Check for updates to Go dependencies",
	Long: `go-check-updates (gcu) for Go modules.

It allows you to list available updates, interactively select them, and upgrade your go.mod file.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := app.Run(app.RunOptions{
			Upgrade:             upgradeFlag,
			Interactive:         verifyFlag,
			Filter:              filterFlag,
			All:                 allFlag,
			Cooldown:            cooldownFlag,
			FormatFlag:          formatFlag,
			ShowVulnerabilities: vulnerabilitiesFlag,
		}, app.Deps{
			Out:            os.Stdout,
			Now:            time.Now,
			GetUpdates:     scanner.GetUpdates,
			UpdatePackages: updater.UpdatePackages,
			StartInteractive: func(direct, indirect, transitive []scanner.Module, opts tui.Options) {
				tui.StartInteractiveGroupedWithOptions(direct, indirect, transitive, opts)
			},
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&upgradeFlag, "upgrade", "u", false, "Upgrade all packages to the latest version")
	rootCmd.Flags().BoolVarP(&verifyFlag, "interactive", "i", false, "Interactive mode")
	rootCmd.Flags().StringVarP(&filterFlag, "filter", "f", "", "Filter packages using regex")
	rootCmd.Flags().BoolVar(&allFlag, "all", false, "Include transitive updates (not listed in go.mod)")
	rootCmd.Flags().IntVarP(&cooldownFlag, "cooldown", "c", 0, "Minimum age (days) for an update to be considered")
	rootCmd.Flags().StringVar(&formatFlag, "format", "", "Output format modifiers: group,lines,time (comma-delimited)")
	rootCmd.Flags().BoolVarP(&vulnerabilitiesFlag, "vulnerabilities", "v", false, "Show vulnerability counts for current and updated versions")
}
