// Package cli implements the MCM command-line contract.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/futuretea/mcm/internal/app"
	"github.com/futuretea/mcm/internal/manifest"
)

// Run executes MCM with an injected process home and I/O streams.
func Run(args []string, userHome string, in io.Reader, out, errOut io.Writer) int {
	options, command, rest, err := parseGlobal(args)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if command == "" {
		fmt.Fprintln(errOut, "a command is required")
		return 2
	}
	if command == "recover" && options.config != "" {
		fmt.Fprintln(errOut, "recover does not accept --config")
		return 2
	}
	if options.home == "" && !filepath.IsAbs(userHome) {
		fmt.Fprintln(errOut, "HOME must be absolute when --home is not set")
		return 2
	}
	location := manifest.NewLocation(userHome, options.home, options.config)

	switch command {
	case "init":
		if len(rest) != 0 {
			fmt.Fprintln(errOut, "init accepts no command flags")
			return 2
		}
		if err := location.Init(); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, "initialized")
		return 0
	case "validate":
		if len(rest) != 0 {
			fmt.Fprintln(errOut, "validate accepts no command flags")
			return 2
		}
		if _, err := location.Load(); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, "valid")
		return 0
	case "server":
		return runServer(rest, location, in, out, errOut)
	case "plan", "apply", "status":
		return runTargetCommand(command, rest, userHome, location, in, out, errOut)
	case "recover":
		if len(rest) != 0 {
			fmt.Fprintln(errOut, "recover accepts no command flags")
			return 2
		}
		if err := app.New(userHome, location).Recover(); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, "recovered")
		return 0
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", command)
		return 2
	}
}

type globalOptions struct {
	home   string
	config string
}

func parseGlobal(args []string) (globalOptions, string, []string, error) {
	options := globalOptions{}
	index := 0
	for index < len(args) {
		argument := args[index]
		if argument != "--home" && argument != "--config" {
			break
		}
		if index+1 >= len(args) {
			return options, "", nil, fmt.Errorf("%s requires a value", argument)
		}
		value := args[index+1]
		if !filepath.IsAbs(value) {
			return options, "", nil, fmt.Errorf("%s must be absolute", argument)
		}
		if argument == "--home" {
			options.home = value
		} else {
			options.config = value
		}
		index += 2
	}
	if index == len(args) {
		return options, "", nil, nil
	}
	return options, args[index], args[index+1:], nil
}

func runTargetCommand(command string, args []string, userHome string, location manifest.Location, in io.Reader, out, errOut io.Writer) int {
	targets, path, yes, err := parseTargetFlags(args, command == "apply")
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	application := app.New(userHome, location)
	switch command {
	case "plan":
		items, err := application.Plan(targets, path)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		printPlan(out, items)
		return 0
	case "apply":
		fmt.Fprintln(out, "warning: external writers can change a target after final verification and before rename")
		var preview []app.PlanItem
		if !yes {
			if !isTerminal(in) {
				fmt.Fprintln(errOut, "apply requires --yes in non-interactive mode")
				return 2
			}
			items, err := application.Plan(targets, path)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return 1
			}
			printPlan(out, items)
			fmt.Fprint(out, "Apply these changes? [yes/no]: ")
			answer, err := bufio.NewReader(in).ReadString('\n')
			if err != nil && len(answer) == 0 {
				fmt.Fprintln(errOut, "confirmation cancelled")
				return 1
			}
			if strings.TrimSpace(answer) != "yes" {
				fmt.Fprintln(errOut, "confirmation cancelled")
				return 1
			}
			preview = items
		}
		var items []app.PlanItem
		var err error
		if preview != nil {
			items, err = application.ApplyPlanned(preview)
		} else {
			items, err = application.Apply(targets, path)
		}
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		printPlan(out, items)
		for _, item := range items {
			if item.Target != "mcpc" {
				continue
			}
			for _, change := range item.Changes {
				if change.Action == "remove" {
					continue
				}
				fmt.Fprintf(out, "mcpc connect %s\n", shellQuote(item.Path+":"+change.Name))
			}
		}
		return 0
	case "status":
		items, err := application.Status(targets, path)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		for _, item := range items {
			fmt.Fprintf(out, "%s  %s  %s\n", item.Target, item.State, item.Path)
			switch item.Target {
			case "qoder-ide":
				fmt.Fprintln(out, "  file-only: IDE load state is not verified")
			case "mcp-cli":
				fmt.Fprintf(out, "  file-only: use mcp-cli --config %s\n", shellQuote(item.Path))
			}
		}
		return 0
	default:
		return 2
	}
}

func isTerminal(input io.Reader) bool {
	file, ok := input.(interface{ Stat() (os.FileInfo, error) })
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func parseTargetFlags(args []string, allowYes bool) ([]string, string, bool, error) {
	var targets []string
	path := ""
	yes := false
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--target":
			if index+1 >= len(args) {
				return nil, "", false, fmt.Errorf("--target requires a value")
			}
			targets = append(targets, args[index+1])
			index++
		case "--path":
			if index+1 >= len(args) || path != "" {
				return nil, "", false, fmt.Errorf("--path requires one value")
			}
			path = args[index+1]
			if !filepath.IsAbs(path) {
				return nil, "", false, fmt.Errorf("--path must be absolute")
			}
			index++
		case "--yes":
			if !allowYes {
				return nil, "", false, fmt.Errorf("--yes is only valid for apply")
			}
			yes = true
		default:
			return nil, "", false, fmt.Errorf("unknown command flag %q", args[index])
		}
	}
	return targets, path, yes, nil
}

func printPlan(out io.Writer, items []app.PlanItem) {
	for _, item := range items {
		fmt.Fprintf(out, "%s  %s\n", item.Target, item.Path)
		for _, change := range item.Changes {
			fmt.Fprintf(out, "  %s: %s\n", change.Action, change.Name)
		}
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
