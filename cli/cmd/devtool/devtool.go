package devtool

import (
	"github.com/fridencao/stardata/cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func DevtoolCmd(ch *cmdutil.Helper) *cobra.Command {
	internalGroupID := ""
	devtoolCmd := &cobra.Command{
		Use:   "devtool",
		Short: "Utilities for developing StarData",
		Example: `  stardata devtool start cloud
  stardata devtool seed cloud
  stardata devtool start cloud --reset
  stardata devtool start cloud --except runtime
  stardata devtool start cloud --only admin,deps
  stardata devtool start local
  stardata devtool start local --reset
  stardata devtool switch-env stage
  stardata devtool dotenv upload cloud`,
		Hidden:  !ch.IsDev(),
		GroupID: internalGroupID,
	}

	devtoolCmd.AddCommand(StartCmd(ch))
	devtoolCmd.AddCommand(SeedCmd(ch))
	devtoolCmd.AddCommand(DotenvCmd(ch))
	devtoolCmd.AddCommand(SwitchEnvCmd(ch))
	devtoolCmd.AddCommand(SubscriptionCmd(ch))

	return devtoolCmd
}
