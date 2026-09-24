package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const environmentGroupsFile = "groups.yaml"

type environmentGroupFile struct {
	Groups map[string][]string `yaml:"groups"`
}

var configGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage named environment groups",
}

var configGroupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured environment groups",
	RunE: func(_ *cobra.Command, _ []string) error {
		groups, err := loadEnvironmentGroups()
		if err != nil {
			return err
		}
		if len(groups) == 0 {
			fmt.Println("No environment groups configured")
			return nil
		}

		names := make([]string, 0, len(groups))
		for name := range groups {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Printf("%s\t%d environments\n", name, len(groups[name]))
		}
		return nil
	},
}

var configGroupsShowCmd = &cobra.Command{
	Use:   "show <group>",
	Short: "Show the environments in a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		groups, err := loadEnvironmentGroups()
		if err != nil {
			return err
		}
		members, ok := groups[args[0]]
		if !ok {
			return fmt.Errorf("environment group %q not found in %s", args[0], filepath.Join(configDir(), environmentGroupsFile))
		}

		fmt.Printf("%s:\n", args[0])
		for _, member := range members {
			fmt.Printf("  - %s\n", member)
		}
		return nil
	},
}

var selectedEnvironments []string

func loadEnvironmentGroups() (map[string][]string, error) {
	path := filepath.Join(configDir(), environmentGroupsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string][]string{}, nil
		}
		return nil, fmt.Errorf("reading environment groups from %s: %w", path, err)
	}

	var file environmentGroupFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing environment groups in %s: %w", path, err)
	}
	if file.Groups == nil {
		return map[string][]string{}, nil
	}

	groups := make(map[string][]string, len(file.Groups))
	for name, members := range file.Groups {
		if !validEnvName.MatchString(name) {
			return nil, fmt.Errorf("invalid environment group name %q in %s", name, path)
		}
		if len(members) == 0 {
			return nil, fmt.Errorf("environment group %q in %s has no members", name, path)
		}

		seen := make(map[string]bool, len(members))
		for _, member := range members {
			member = strings.TrimSpace(member)
			if !validEnvName.MatchString(member) {
				return nil, fmt.Errorf("invalid environment %q in group %q (%s)", member, name, path)
			}
			if seen[member] {
				continue
			}
			seen[member] = true
			groups[name] = append(groups[name], member)
		}
	}

	return groups, nil
}

func resolveEnvironmentTargets(envValue string, args []string) ([]string, error) {
	references := args
	if len(references) == 0 {
		if envValue == "" {
			return nil, nil
		}
		references = strings.Split(envValue, ",")
	}

	groups, err := loadEnvironmentGroups()
	if err != nil {
		return nil, err
	}

	targets := make([]string, 0, len(references))
	seen := make(map[string]bool)
	for _, reference := range references {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return nil, fmt.Errorf("environment target cannot be empty")
		}

		members, isGroup := groups[reference]
		if !isGroup {
			members = []string{reference}
		}
		for _, member := range members {
			if !validEnvName.MatchString(member) {
				return nil, fmt.Errorf("invalid environment name %q: must be alphanumeric with hyphens or underscores", member)
			}
			if !seen[member] {
				seen[member] = true
				targets = append(targets, member)
			}
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("environment targets resolved to an empty list")
	}

	return targets, nil
}

func supportsEnvironmentGroups(cmd *cobra.Command) bool {
	switch cmd.CommandPath() {
	case "succulent-cli delete", "succulent-cli reprovision",
		"succulent-cli sno provision", "succulent-cli sno kubeconfig",
		"succulent-cli ztp provision", "succulent-cli ztp kubeconfig",
		"succulent-cli hypershift provision", "succulent-cli hypershift kubeconfig",
		"succulent-cli kubeconfig fetch":
		return true
	default:
		return false
	}
}

func currentEnvironmentTargets() []string {
	if len(selectedEnvironments) > 0 {
		return selectedEnvironments
	}
	if envName != "" {
		return []string{envName}
	}
	return nil
}

func environmentDestination(dest, target string, targets []string) (string, error) {
	if dest == "" {
		return "", nil
	}
	if err := validateBatchDestination(dest, targets); err != nil {
		return "", err
	}
	return strings.ReplaceAll(dest, "{env}", target), nil
}

func validateBatchDestination(dest string, targets []string) error {
	if dest != "" && len(targets) > 1 && !strings.Contains(dest, "{env}") {
		return fmt.Errorf("--dest must include {env} when using multiple environments (for example: ./kubeconfigs/{env}.yaml)")
	}
	return nil
}

type environmentOutcome struct {
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

func runForEnvironmentTargets(action func(string) (string, error), singleResult func(string, string) error, dryRun bool) error {
	targets := currentEnvironmentTargets()
	if len(targets) == 0 {
		return fmt.Errorf("no environments selected")
	}

	if len(targets) == 1 {
		message, err := action(targets[0])
		if err != nil || dryRun || singleResult == nil {
			return err
		}
		return singleResult(targets[0], message)
	}

	fmt.Fprintf(os.Stderr, "Operating on environments: %s\n", strings.Join(targets, ", "))
	outcomes := make([]environmentOutcome, 0, len(targets))
	failures := 0
	for _, target := range targets {
		message, err := action(target)
		outcome := environmentOutcome{Environment: target, Message: message}
		if err != nil {
			failures++
			outcome.Status = "failed"
			outcome.Message = err.Error()
		} else {
			outcome.Status = "success"
		}
		outcomes = append(outcomes, outcome)
	}

	if outputFormat == "json" {
		if err := printJSON(outcomes); err != nil {
			return err
		}
	} else {
		for _, outcome := range outcomes {
			mark := "✓"
			if outcome.Status == "failed" {
				mark = "✗"
			}
			fmt.Printf("%s %s: %s\n", mark, outcome.Environment, outcome.Message)
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d environment operations failed", failures, len(targets))
	}
	return nil
}

func init() {
	configGroupsCmd.AddCommand(configGroupsListCmd, configGroupsShowCmd)
}
