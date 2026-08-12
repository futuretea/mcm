package cli

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/futuretea/mcm/internal/manifest"
	"github.com/futuretea/mcm/internal/safeio"
)

type serverInput struct {
	name    string
	command string
	url     string
	args    []string
}

func runServer(args []string, location manifest.Location, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "server requires list, add, or update")
		return 2
	}
	switch args[0] {
	case "--help", "help":
		printServerHelp(out)
		return 0
	case "list":
		if isHelp(args[1:]) {
			printServerHelp(out)
			return 0
		}
		return runServerList(args[1:], location, out, errOut)
	case "add":
		if isHelp(args[1:]) {
			printServerHelp(out)
			return 0
		}
		return runServerWrite(args[1:], location, in, out, errOut, false)
	case "update":
		if isHelp(args[1:]) {
			printServerHelp(out)
			return 0
		}
		return runServerWrite(args[1:], location, in, out, errOut, true)
	default:
		fmt.Fprintln(errOut, "server requires list, add, or update")
		return 2
	}
}

func isHelp(args []string) bool {
	return len(args) == 1 && args[0] == "--help"
}

func printServerHelp(out io.Writer) {
	fmt.Fprint(out, `Usage: mcm server <command> [flags]

Commands:
  add      Add a new server. Fails if the name already exists.
  update   Replace an existing server. Fails if the name does not exist.
  list     List server names.

Server flags:
  --name NAME
  --command COMMAND [--arg ARG ...]
  --url URL
`)
}

func runServerList(args []string, location manifest.Location, out, errOut io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(errOut, "server list accepts no flags")
		return 2
	}
	config, err := location.Load()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	for _, name := range sortedNames(config.Servers) {
		fmt.Fprintln(out, name)
	}
	return 0
}

func runServerWrite(args []string, location manifest.Location, in io.Reader, out, errOut io.Writer, update bool) int {
	command := "add"
	if update {
		command = "update"
	}
	input, err := parseServerInput(args, command)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if input.conflicts() {
		fmt.Fprintf(errOut, "server %s requires --name and exactly one of --command or --url\n", command)
		return 2
	}
	if input.incomplete() && isTerminal(in) {
		input, err = promptMissingServerFields(in, out, input)
		if err != nil {
			fmt.Fprintf(errOut, "server %s cancelled\n", command)
			return 1
		}
	}
	if input.incomplete() || input.conflicts() {
		fmt.Fprintf(errOut, "server %s requires --name and exactly one of --command or --url\n", command)
		return 2
	}
	return saveServer(location, input, out, errOut, update)
}

func parseServerInput(args []string, command string) (serverInput, error) {
	input := serverInput{}
	for index := 0; index < len(args); index++ {
		if index+1 >= len(args) {
			return serverInput{}, fmt.Errorf("%s requires a value", args[index])
		}
		value := args[index+1]
		switch args[index] {
		case "--name":
			input.name = value
		case "--command":
			input.command = value
		case "--url":
			input.url = value
		case "--arg":
			input.args = append(input.args, value)
		default:
			return serverInput{}, fmt.Errorf("unknown server %s flag %q", command, args[index])
		}
		index++
	}
	return input, nil
}

func (input serverInput) incomplete() bool {
	return input.name == "" || (input.command == "" && input.url == "")
}

func (input serverInput) conflicts() bool {
	return (input.command != "" && input.url != "") || (input.url != "" && len(input.args) > 0)
}

func promptMissingServerFields(in io.Reader, out io.Writer, input serverInput) (serverInput, error) {
	reader := bufio.NewReader(in)
	read := func(label string) (string, error) {
		fmt.Fprint(out, label)
		value, err := reader.ReadString('\n')
		if err != nil && len(value) == 0 {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("empty input")
		}
		return value, nil
	}
	transport := ""
	if input.command == "" && input.url == "" {
		var err error
		transport, err = read("transport (stdio/streamable-http): ")
		if err != nil || (transport != manifest.TransportStdio && transport != manifest.TransportStreamableHTTP) {
			return serverInput{}, fmt.Errorf("unsupported transport")
		}
	}
	if input.name == "" {
		name, err := read("name: ")
		if err != nil {
			return serverInput{}, err
		}
		input.name = name
	}
	if transport == manifest.TransportStreamableHTTP {
		url, err := read("url: ")
		if err != nil {
			return serverInput{}, err
		}
		input.url = url
	}
	if input.url != "" || input.command != "" {
		return input, nil
	}
	command, err := read("command: ")
	if err != nil {
		return serverInput{}, err
	}
	input.command = command
	for {
		fmt.Fprint(out, "arg (blank to finish): ")
		value, err := reader.ReadString('\n')
		if err != nil && len(value) == 0 {
			return serverInput{}, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return input, nil
		}
		input.args = append(input.args, value)
	}
}

func saveServer(location manifest.Location, input serverInput, out, errOut io.Writer, update bool) int {
	lock, err := safeio.AcquireLock(filepath.Join(location.Root, "lock"))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer lock.Close()
	config, err := location.Load()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if config.Servers == nil {
		config.Servers = map[string]manifest.Server{}
	}
	_, exists := config.Servers[input.name]
	if update && !exists {
		fmt.Fprintf(errOut, "server %q not found; use \"mcm server add\" to create it\n", input.name)
		return 1
	}
	if !update && exists {
		fmt.Fprintf(errOut, "server %q already exists; use \"mcm server update\" to replace it\n", input.name)
		return 1
	}
	server := manifest.Server{Command: input.command, Args: input.args}
	if input.command != "" {
		server.Transport = manifest.TransportStdio
	} else {
		server.Transport = manifest.TransportStreamableHTTP
		server.URL = input.url
	}
	config.Servers[input.name] = server
	if err := location.Save(config); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintln(out, "saved")
	return 0
}

func sortedNames(values map[string]manifest.Server) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
