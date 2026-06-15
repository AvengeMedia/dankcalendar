package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
)

var ipcCmd = &cobra.Command{
	Use:   "ipc <method> [key=value...]",
	Short: "Send an IPC request to the running dcal daemon",
	Long: `Send an IPC request to the running dcal daemon.

  dcal ipc <method> [key=value...]   invoke a method
  dcal ipc list                      list every method and its params

Method names and param keys complete with the shell (see 'dcal completion').`,
	Args:                  cobra.MinimumNArgs(1),
	ValidArgsFunction:     completeIPC,
	DisableFlagsInUseLine: true,
	RunE: func(_ *cobra.Command, args []string) error {
		method := args[0]
		params, err := parseParams(args[1:])
		if err != nil {
			return err
		}

		socketPath := os.Getenv("DANKCAL_SOCKET")
		if socketPath == "" {
			socketPath, err = ipc.FindRunningSocket()
			if err != nil {
				return fmt.Errorf("dcal daemon not running: %w", err)
			}
		}

		client, err := ipc.Dial(socketPath)
		if err != nil {
			return err
		}
		defer client.Close()

		resp, err := client.Call(ipc.Request{ID: 1, Method: method, Params: params})
		if err != nil {
			return err
		}

		if resp.Error != "" {
			return fmt.Errorf("ipc error: %s", resp.Error)
		}

		out, err := json.MarshalIndent(resp.Result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(out))
		return nil
	},
}

var ipcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every IPC method and its params",
	RunE: func(_ *cobra.Command, _ []string) error {
		if jsonOutput {
			return printJSON(map[string]any{"apiVersion": ipc.APIVersion, "methods": ipc.Methods})
		}
		printIPCList()
		return nil
	},
}

func init() {
	ipcCmd.AddCommand(ipcListCmd)
}

func completeIPC(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return ipc.MethodNames(), cobra.ShellCompDirectiveNoFileComp
	}

	spec, ok := ipc.FindMethod(args[0])
	if !ok {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	used := make(map[string]bool, len(args))
	for _, a := range args[1:] {
		if k, _, found := strings.Cut(a, "="); found {
			used[k] = true
		}
	}

	comps := make([]string, 0, len(spec.Params))
	for _, p := range spec.Params {
		if used[p.Name] {
			continue
		}
		desc := p.Desc
		if p.Required {
			desc = strings.TrimSpace("required " + desc)
		}
		comps = append(comps, fmt.Sprintf("%s=\t%s", p.Name, desc))
	}
	return comps, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func printIPCList() {
	byGroup := make(map[string][]ipc.MethodSpec)
	groups := make([]string, 0)
	for _, m := range ipc.Methods {
		if _, seen := byGroup[m.Group]; !seen {
			groups = append(groups, m.Group)
		}
		byGroup[m.Group] = append(byGroup[m.Group], m)
	}
	sort.Strings(groups)

	for _, group := range groups {
		fmt.Printf("%s\n", group)
		for _, m := range byGroup[group] {
			fmt.Printf("  %-32s %s\n", m.Name, m.Desc)
			for _, p := range m.Params {
				marker := " "
				if p.Required {
					marker = "*"
				}
				line := fmt.Sprintf("      %s%s", marker, p.Name)
				if p.Desc != "" {
					line = fmt.Sprintf("%-32s %s", line, p.Desc)
				}
				fmt.Println(line)
			}
		}
		fmt.Println()
	}
	fmt.Println("* = required   |   usage: dcal ipc <method> key=value ...")
}

func parseParams(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, nil
	}
	params := make(map[string]any, len(args))
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("expected key=value, got %q", arg)
		}
		params[key] = value
	}
	return params, nil
}
