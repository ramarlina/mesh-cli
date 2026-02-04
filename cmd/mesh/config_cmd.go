package main

import (
	"sort"

	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configLsCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local settings",
	Long:  "View and modify CLI configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var configLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all config settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		settings, err := config.List()
		if err != nil {
			return out.Error(err)
		}

		if out.IsJSON() {
			return out.Success(settings)
		}

		// Sort keys for consistent output
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		headers := []string{"Key", "Value"}
		rows := [][]string{}

		for _, k := range keys {
			v := settings[k]
			if v == "" {
				v = "(not set)"
			}
			rows = append(rows, []string{k, v})
		}

		return out.Table(headers, rows)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		key := args[0]
		value, err := config.Get(key)
		if err != nil {
			return out.Error(err)
		}

		if out.IsJSON() {
			return out.Success(map[string]string{key: value})
		}

		if out.IsRaw() {
			out.Println(value)
		} else {
			out.Printf("%s: %s\n", key, value)
		}

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := getOutputPrinter()

		key := args[0]
		value := args[1]

		if err := config.Set(key, value); err != nil {
			return out.Error(err)
		}

		if out.IsJSON() {
			return out.Success(map[string]string{key: value})
		}

		out.Printf("✓ Set %s = %s\n", key, value)
		return nil
	},
}
