package sudo

import (
	"github.com/fridencao/stardata/cli/cmd/sudo/annotations"
	"github.com/fridencao/stardata/cli/cmd/sudo/billing"
	"github.com/fridencao/stardata/cli/cmd/sudo/embed"
	"github.com/fridencao/stardata/cli/cmd/sudo/org"
	"github.com/fridencao/stardata/cli/cmd/sudo/project"
	"github.com/fridencao/stardata/cli/cmd/sudo/quota"
	"github.com/fridencao/stardata/cli/cmd/sudo/runtime"
	"github.com/fridencao/stardata/cli/cmd/sudo/superuser"
	"github.com/fridencao/stardata/cli/cmd/sudo/user"
	"github.com/fridencao/stardata/cli/cmd/sudo/virtualfiles"
	"github.com/fridencao/stardata/cli/cmd/sudo/whitelist"
	"github.com/fridencao/stardata/cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func SudoCmd(ch *cmdutil.Helper) *cobra.Command {
	internalGroupID := ""
	sudoCmd := &cobra.Command{
		Use:     "sudo",
		Short:   "sudo commands for superusers",
		Hidden:  !ch.IsDev(),
		GroupID: internalGroupID,
	}
	sudoCmd.AddCommand(lookupCmd(ch))
	sudoCmd.AddCommand(embed.EmbedCmd(ch))
	sudoCmd.AddCommand(org.OrgCmd(ch))
	sudoCmd.AddCommand(project.ProjectCmd(ch))
	sudoCmd.AddCommand(user.UserCmd(ch))
	sudoCmd.AddCommand(superuser.SuperuserCmd(ch))
	sudoCmd.AddCommand(billing.BillingCmd(ch))
	sudoCmd.AddCommand(quota.QuotaCmd(ch))
	sudoCmd.AddCommand(whitelist.WhitelistCmd(ch))
	sudoCmd.AddCommand(annotations.AnnotationsCmd(ch))
	sudoCmd.AddCommand(cloneCmd(ch))
	sudoCmd.AddCommand(runtime.RuntimeCmd(ch))
	sudoCmd.AddCommand(virtualfiles.VirtualFilesCmd(ch))

	return sudoCmd
}
