package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var version = "dev"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

type Config struct {
	dir       string
	envFile   string
	sample    string
	copy      bool
	recursive bool
	dryRun    bool
	version   bool
}

type EnvEntry struct {
	Key        string
	Value      string
	HasValue   bool
	Comments   []string
	LineNumber int
	Raw        string
}

type EnvState int

const (
	StateMissing EnvState = iota
	StateUnconfigured
	StateConfigured
)

type ProjectResult struct {
	Path         string
	Missing      []EnvEntry
	Unconfigured []EnvEntry
	Configured   []EnvEntry
	Skipped      bool
}

func main() {
	config := parseFlags()

	if config.dir == "" {
		printUsage()
		return
	}

	if config.version {
		fmt.Printf("envcheck %s\n", version)
		return
	}

	if config.recursive {
		if err := scanProjects(config); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(2)
		}
		return
	}

	result, err := checkProject(config.dir, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
		os.Exit(2)
	}

	printProjectResult(result)

	if len(result.Missing) == 0 && len(result.Unconfigured) == 0 {
		return
	}

	if config.dryRun {
		fmt.Printf("\n%s%sDry run:%s no files were changed.\n",
			colorBold, colorCyan, colorReset)
		return
	}

	if len(result.Missing) > 0 {
		if config.copy {
			if err := copyMissing(config.dir, config.envFile, result.Missing); err != nil {
				fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
				os.Exit(2)
			}

			fmt.Printf("\n%s✓ Added %d missing variable(s) to %s%s\n",
				colorGreen, len(result.Missing), config.envFile, colorReset)

			printUnconfiguredWarning(result.Missing, result.Unconfigured)
			return
		}

		choice, err := promptAction()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(2)
		}

		switch choice {
		case "c":
			if err := copyMissing(config.dir, config.envFile, result.Missing); err != nil {
				fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
				os.Exit(2)
			}

			fmt.Printf("\n%s✓ Added %d missing variable(s) to %s%s\n",
				colorGreen, len(result.Missing), config.envFile, colorReset)

			printUnconfiguredWarning(result.Missing, result.Unconfigured)

		case "i":
			fmt.Printf("\n%sNo changes made.%s\n", colorCyan, colorReset)
			printUnconfiguredWarning(nil, result.Unconfigured)

		default:
			fmt.Printf("\n%sNo changes made.%s\n", colorCyan, colorReset)
		}

		return
	}

	// There are no missing keys, but there are empty/unconfigured keys.
	printUnconfiguredWarning(nil, result.Unconfigured)
}

func parseFlags() Config {
	var config Config

	flag.Usage = printUsage

	flag.StringVar(&config.dir, "d", "", "directory to check")
	flag.StringVar(&config.envFile, "e", ".env", "environment file name")
	flag.StringVar(&config.sample, "s", ".env_sample", "sample environment file name")
	flag.BoolVar(&config.copy, "c", false, "copy missing variables automatically")
	flag.BoolVar(&config.recursive, "r", false, "scan immediate subdirectories")
	flag.BoolVar(&config.dryRun, "n", false, "dry run; do not modify files")
	flag.BoolVar(&config.version, "v", false, "show version")

	flag.Parse()

	return config
}

func printUsage() {
	fmt.Printf(`envcheck - check .env files against .env_sample

Usage:
  envcheck [options]

Options:
  -d <dir>     Directory to check       (default: .)
  -e <file>    Environment file          (default: .env)
  -s <file>    Sample environment file   (default: .env_sample)
  -c           Copy missing variables automatically
  -r           Scan immediate subdirectories
  -n           Dry run; do not modify files
  -v           Show version
  -h           Show this help

Examples:
  envcheck
  envcheck -d /opt/deployment/myapp
  envcheck -d /opt/deployment -r
  envcheck -d /opt/deployment -r -c
  envcheck -d /opt/deployment -r -n
  envcheck -d . -e .env.production -s .env.example

Behavior:
  Missing key:
    The key exists in .env_sample but not in .env.

  Unconfigured key:
    The key exists in .env but has an empty value.

  Configured key:
    The key exists in .env and has a non-empty value.

Important:
  - Sample values are NEVER copied.
  - Existing values are NEVER changed.
  - Empty values are reported on every run.
  - Missing keys are added only once.
  - Missing keys follow .env_sample order.
  - Associated comments are copied.
  - A backup is created before modifying .env.
`)
}

func scanProjects(config Config) error {
	entries, err := os.ReadDir(config.dir)
	if err != nil {
		return fmt.Errorf("cannot read directory %q: %w", config.dir, err)
	}

	foundProject := false
	totalMissing := 0
	totalUnconfigured := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(config.dir, entry.Name())

		envPath := filepath.Join(projectDir, config.envFile)
		samplePath := filepath.Join(projectDir, config.sample)

		if !fileExists(envPath) || !fileExists(samplePath) {
			continue
		}

		foundProject = true

		projectConfig := config
		projectConfig.dir = projectDir

		result, err := checkProject(projectDir, projectConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"%sWarning:%s %s: %v\n",
				colorYellow,
				colorReset,
				projectDir,
				err,
			)
			continue
		}

		fmt.Println()
		printProjectResult(result)

		totalMissing += len(result.Missing)
		totalUnconfigured += len(result.Unconfigured)

		if config.dryRun {
			continue
		}

		if len(result.Missing) == 0 {
			continue
		}

		if config.copy {
			if err := copyMissing(projectDir, config.envFile, result.Missing); err != nil {
				fmt.Fprintf(os.Stderr,
					"%sError:%s %s: %v\n",
					colorRed,
					colorReset,
					projectDir,
					err,
				)
				continue
			}

			fmt.Printf(
				"%s✓ Added %d missing variable(s)%s\n",
				colorGreen,
				len(result.Missing),
				colorReset,
			)

			printUnconfiguredWarning(result.Missing, result.Unconfigured)
			continue
		}

		choice, err := promptAction()
		if err != nil {
			return err
		}

		if choice == "c" {
			if err := copyMissing(projectDir, config.envFile, result.Missing); err != nil {
				fmt.Fprintf(os.Stderr,
					"%sError:%s %v\n",
					colorRed,
					colorReset,
					err,
				)
				continue
			}

			fmt.Printf(
				"%s✓ Added %d missing variable(s)%s\n",
				colorGreen,
				len(result.Missing),
				colorReset,
			)

			printUnconfiguredWarning(result.Missing, result.Unconfigured)
		} else {
			fmt.Printf("%sNo changes made.%s\n", colorCyan, colorReset)
			printUnconfiguredWarning(nil, result.Unconfigured)
		}
	}

	if !foundProject {
		return fmt.Errorf(
			"no projects found under %q containing both %s and %s",
			config.dir,
			config.envFile,
			config.sample,
		)
	}

	fmt.Println()
	fmt.Printf(
		"%sSummary:%s %d missing, %d unconfigured\n",
		colorBold,
		colorReset,
		totalMissing,
		totalUnconfigured,
	)

	return nil
}

func checkProject(dir string, config Config) (ProjectResult, error) {
	envPath := filepath.Join(dir, config.envFile)
	samplePath := filepath.Join(dir, config.sample)

	if !fileExists(envPath) {
		return ProjectResult{}, fmt.Errorf(
			"%s does not exist",
			envPath,
		)
	}

	if !fileExists(samplePath) {
		return ProjectResult{}, fmt.Errorf(
			"%s does not exist",
			samplePath,
		)
	}

	sampleEntries, err := parseEnvFile(samplePath)
	if err != nil {
		return ProjectResult{}, fmt.Errorf(
			"cannot parse %s: %w",
			samplePath,
			err,
		)
	}

	envEntries, err := parseEnvFile(envPath)
	if err != nil {
		return ProjectResult{}, fmt.Errorf(
			"cannot parse %s: %w",
			envPath,
			err,
		)
	}

	envMap := make(map[string]EnvEntry)

	for _, entry := range envEntries {
		if entry.Key == "" {
			continue
		}

		envMap[entry.Key] = entry
	}

	result := ProjectResult{
		Path: dir,
	}

	for _, sampleEntry := range sampleEntries {
		if sampleEntry.Key == "" {
			continue
		}

		envEntry, exists := envMap[sampleEntry.Key]

		if !exists {
			result.Missing = append(result.Missing, sampleEntry)
			continue
		}

		if !hasConfiguredValue(envEntry.Value) {
			result.Unconfigured = append(result.Unconfigured, sampleEntry)
			continue
		}

		result.Configured = append(result.Configured, sampleEntry)
	}

	return result, nil
}

func parseEnvFile(path string) ([]EnvEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []EnvEntry
	var pendingComments []string

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			pendingComments = nil
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			pendingComments = append(pendingComments, line)
			continue
		}

		key, value, ok := parseEnvLine(line)

		if !ok {
			pendingComments = nil
			continue
		}

		entries = append(entries, EnvEntry{
			Key:        key,
			Value:      value,
			HasValue:   true,
			Comments:   append([]string(nil), pendingComments...),
			LineNumber: lineNumber,
			Raw:        line,
		})

		pendingComments = nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func parseEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	equalIndex := strings.Index(trimmed, "=")

	if equalIndex < 0 {
		// Support KEY without "=" as a key, but treat it as unconfigured.
		key := strings.TrimSpace(trimmed)

		if key == "" {
			return "", "", false
		}

		return key, "", true
	}

	key := strings.TrimSpace(trimmed[:equalIndex])
	value := strings.TrimSpace(trimmed[equalIndex+1:])

	if key == "" {
		return "", "", false
	}

	return key, value, true
}

func hasConfiguredValue(value string) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return false
	}

	// Treat empty quoted values as unconfigured.
	if value == `""` || value == `''` {
		return false
	}

	// Treat whitespace inside quotes as unconfigured.
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]

		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') {
			inner := strings.TrimSpace(value[1 : len(value)-1])

			if inner == "" {
				return false
			}
		}
	}

	return true
}

func copyMissing(dir, envFile string, missing []EnvEntry) error {
	if len(missing) == 0 {
		return nil
	}

	envPath := filepath.Join(dir, envFile)

	original, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", envPath, err)
	}

	backupPath, err := createBackup(envPath, original)
	if err != nil {
		return fmt.Errorf("cannot create backup: %w", err)
	}

	fmt.Printf(
		"%sBackup created:%s %s\n",
		colorCyan,
		colorReset,
		backupPath,
	)

	var builder strings.Builder

	builder.Write(original)

	// Ensure there is a newline before the generated section.
	if len(original) > 0 && !strings.HasSuffix(string(original), "\n") {
		builder.WriteString("\n")
	}

	builder.WriteString("\n")
	builder.WriteString("# ------------------------------------------------------------\n")
	builder.WriteString("# !!! CONFIGURATION REQUIRED !!!\n")
	builder.WriteString("# The following values were added by envcheck.\n")
	builder.WriteString("# Replace the empty values before starting the application.\n")
	builder.WriteString("# ------------------------------------------------------------\n")

	for _, entry := range missing {
		builder.WriteString("\n")

		for _, comment := range entry.Comments {
			builder.WriteString(comment)
			builder.WriteString("\n")
		}

		builder.WriteString("# Added by envcheck - VALUE REQUIRED\n")
		builder.WriteString(entry.Key)
		builder.WriteString("=\n")
	}

	builder.WriteString("\n")
	builder.WriteString("# ------------------------------------------------------------\n")
	builder.WriteString("# !!! END OF CONFIGURATION REQUIRED !!!\n")
	builder.WriteString("# ------------------------------------------------------------\n")

	if err := os.WriteFile(
		envPath,
		[]byte(builder.String()),
		0600,
	); err != nil {
		return fmt.Errorf("cannot update %s: %w", envPath, err)
	}

	return nil
}

func createBackup(path string, content []byte) (string, error) {
	backupPath := path + ".envcheck-backup"

	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return "", err
	}

	return backupPath, nil
}

func printProjectResult(result ProjectResult) {
	fmt.Printf(
		"\n%sProject:%s %s\n",
		colorBold,
		colorReset,
		result.Path,
	)

	if len(result.Missing) == 0 && len(result.Unconfigured) == 0 {
		fmt.Printf(
			"%s✓ All environment variables are configured.%s\n",
			colorGreen,
			colorReset,
		)
		return
	}

	if len(result.Missing) > 0 {
		fmt.Printf(
			"\n%s%s✗ Missing (%d):%s\n",
			colorBold,
			colorRed,
			len(result.Missing),
			colorReset,
		)

		for _, entry := range result.Missing {
			fmt.Printf("  %s%s%s\n",
				colorRed,
				entry.Key,
				colorReset,
			)
		}
	}

	if len(result.Unconfigured) > 0 {
		fmt.Printf(
			"\n%s%s⚠ Unconfigured (%d):%s\n",
			colorBold,
			colorYellow,
			len(result.Unconfigured),
			colorReset,
		)

		for _, entry := range result.Unconfigured {
			fmt.Printf("  %s%s%s\n",
				colorYellow,
				entry.Key,
				colorReset,
			)
		}
	}
}

func printUnconfiguredWarning(added []EnvEntry, existing []EnvEntry) {
	total := len(added) + len(existing)

	if total == 0 {
		fmt.Printf(
			"\n%s✓ All environment variables are configured.%s\n",
			colorGreen,
			colorReset,
		)
		return
	}

	fmt.Printf(
		"\n%s%s⚠ CONFIGURATION REQUIRED%s\n",
		colorBold,
		colorYellow,
		colorReset,
	)

	fmt.Printf(
		"%s%d variable(s) still require a value before the application is started.%s\n",
		colorYellow,
		total,
		colorReset,
	)

	if total > 0 {
		fmt.Println()

		for _, entry := range added {
			fmt.Printf(
				"  %s%s%s\n",
				colorYellow,
				entry.Key,
				colorReset,
			)
		}

		for _, entry := range existing {
			fmt.Printf(
				"  %s%s%s\n",
				colorYellow,
				entry.Key,
				colorReset,
			)
		}
	}
}

func promptAction() (string, error) {
	fmt.Printf(
		"\n%sWhat do you want to do?%s\n\n",
		colorBold,
		colorReset,
	)

	fmt.Println("  [c] Copy missing keys")
	fmt.Println("  [i] Ignore")

	fmt.Printf("\nChoice [c/i]: ")

	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "c", "i":
		return input, nil
	default:
		return "", errors.New("invalid choice; please enter c or i")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)

	if err != nil {
		return false
	}

	return !info.IsDir()
}

func init() {
	// Keep color output useful on terminals, but avoid ANSI escape
	// sequences on Windows where they may not be supported.
	if runtime.GOOS == "windows" {
		// The variables are constants, so there is intentionally no
		// runtime color switch here. Windows Terminal and modern
		// Windows consoles generally support ANSI output.
	}
}

// Keep the compiler aware that sort is intentionally available for
// future deterministic directory handling.
var _ = sort.Strings
