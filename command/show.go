package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/terraform"
)

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
}

func (c *ShowCommand) Run(args []string) int {
	var moduleDepth int

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := flag.NewFlagSet("show", flag.ContinueOnError)
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The show command expects at most one argument with the path\n" +
				"to a Terraform state or plan file.\n")
		cmdFlags.Usage()
		return 1
	}

	var planErr, stateErr error
	var path string
	var plan *terraform.Plan
	var state *terraform.State
	if len(args) > 0 {
		path = args[0]
		f, err := os.Open(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading file: %s", err))
			return 1
		}
		defer f.Close()

		plan, err = terraform.ReadPlan(f)
		if err != nil {
			if _, err := f.Seek(0, 0); err != nil {
				c.Ui.Error(fmt.Sprintf("Error reading file: %s", err))
				return 1
			}

			plan = nil
			planErr = err
		}
		if plan == nil {
			state, err = terraform.ReadState(f)
			if err != nil {
				stateErr = err
			}
		}
	} else {
		// Load the backend
		b, err := c.Backend(nil)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
			return 1
		}

		env := c.Workspace()

		// Get the state
		stateStore, err := b.State(env)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		if err := stateStore.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		state = stateStore.State()
		if state == nil {
			c.Ui.Output("No state.")
			return 0
		}
	}

	if plan == nil && state == nil {
		c.Ui.Error(fmt.Sprintf(
			"Terraform couldn't read the given file as a state or plan file.\n"+
				"The errors while attempting to read the file as each format are\n"+
				"shown below.\n\n"+
				"State read error: %s\n\nPlan read error: %s",
			stateErr,
			planErr))
		return 1
	}

	if plan != nil {
		dispPlan := format.NewPlan(plan)
		c.Ui.Output(dispPlan.Format(c.Colorize()))
		return 0
	}

	c.Ui.Output(format.State(&format.StateOpts{
		State:       state,
		Color:       c.Colorize(),
		ModuleDepth: moduleDepth,
	}))
	return 0
}

func (c *ShowCommand) Help() string {
	helpText := `
Usage: terraform show [options] [path]

  Reads and outputs a Terraform state or plan file in a human-readable
  form. If no path is specified, the current state will be shown.

Options:

  -module-depth=n     Specifies the depth of modules to show in the output.
                      By default this is -1, which will expand all.

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *ShowCommand) Synopsis() string {
	return "Inspect Terraform state or plan"
}
