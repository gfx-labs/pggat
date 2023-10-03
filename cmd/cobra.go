package caddycmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "pggat",
	Long: `
	pggat uses caddy
`,
	Example: `  $ pggat run
  `,

	// kind of annoying to have all the help text printed out if
	// caddy has an error provisioning its modules, for instance...
	SilenceUsage: true,
}

const fullDocsFooter = ``

func init() {
	rootCmd.SetHelpTemplate(rootCmd.HelpTemplate() + "\n" + fullDocsFooter + "\n")
}

func caddyCmdToCobra(caddyCmd Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   caddyCmd.Name,
		Short: caddyCmd.Short,
		Long:  caddyCmd.Long,
	}
	if caddyCmd.CobraFunc != nil {
		caddyCmd.CobraFunc(cmd)
	} else {
		cmd.RunE = WrapCommandFuncForCobra(caddyCmd.Func)
		cmd.Flags().AddGoFlagSet(caddyCmd.Flags)
	}
	return cmd
}

// WrapCommandFuncForCobra wraps a Caddy CommandFunc for use
// in a cobra command's RunE field.
func WrapCommandFuncForCobra(f CommandFunc) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		_, err := f(Flags{cmd.Flags()})
		return err
	}
}
