package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bhakha-services/naagmani-cli/cmd"
)

// Version is the single source of truth for Naagmani CLI version.
const Version = "1.0.0"

// Deterministic exit codes for CI/CD and automation pipelines
const (
	ExitSuccess    = 0 // Command completed successfully
	ExitInvalidArg = 1 // Invalid flags, missing parameters, or usage error
	ExitAuthError  = 2 // Authentication failure, expired session, or unauthorized
	ExitValError   = 3 // Manifest, schema, or policy validation failure
	ExitNetError   = 4 // Network connectivity failure or Cloud API unreachable
)

func exitWithError(code int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(code)
}

func printHelp() {
	fmt.Printf(`Naagmani CLI v%s - Primary Developer Entry Point for Naagmani AI OS

Usage:
  naagmani [command] [flags]

Core Commands:
  help                   Show help for naagmani
  version                Print the CLI version
  doctor                 Diagnose system prerequisites, runtimes, and connectivity

Authentication & Identity:
  login                  Authenticate with Naagmani Cloud
  logout                 Purge locally stored credentials
  whoami                 Display active authenticated identity and tenant context

Project & Configuration:
  init [name]            Initialize a canonical naagmani.yaml project configuration
  config [get|set|list]  Manage local CLI configuration values

Platform Lifecycle & Deployment:
  platform install       Deploy Naagmani P1 platform on Docker/Compose
  platform start/stop    Start or stop platform containers
  platform status        Inspect health and container status (--json)
  platform logs          Stream container logs (-f, --service os|cloud|portal)
  platform update        Perform automated platform update with preflight backup
  platform rollback      Revert to previous known-good release version
  platform backup        Create snapshot backup of 'naagmani' and 'naagmani_cloud' DBs
  platform restore       Restore databases from a snapshot backup
  platform uninstall     Tear down platform containers (--purge-data)

Plugin Lifecycle:
  plugin create <name>   Create a new plugin project (--language go|node|python)
  plugin validate [dir]  Validate plugin.json manifest against protocol rules
  plugin dev [dir]       Build, pre-flight check, and link plugin for local OS testing
  plugin build [dir]     Compile plugin for target platform and calculate SHA-256
  plugin package [dir]   Create distributable .tar.gz archive and checksum
  plugin publish [dir]   Build, package, and publish plugin version to Naagmani Cloud
  plugin install <name>  Install or set desired plugin version for environment
  plugin list / versions List published versions for a plugin
  plugin update <name>   Update desired version of an installed plugin
  plugin rollback <name> Roll back an installed plugin to prior version
  plugin info <name>     Inspect plugin metadata, capabilities, and dependencies
  plugin health <name>   Inspect runtime health status and diagnostics
  plugin metrics <name>  Inspect invocation counters and execution telemetry

Marketplace:
  marketplace search <q> Search the Naagmani Cloud plugin catalog
  marketplace info <name> Inspect marketplace plugin details and capabilities
  marketplace install <name> Install a plugin from the marketplace into active environment

Enterprise Governance & FinOps:
  policy list|get|create|update|enable|disable|rollback|simulate
  usage / cost / budget / quota

Flags:
  -h, --help             Show help
  -v, --version          Show version

Environment Variables:
  NAAGMANI_CLOUD_URL     Naagmani Cloud control plane endpoint (default: http://localhost:8081)
  NAAGMANI_OS_URL        Naagmani OS endpoint (default: http://localhost:8080)
  NAAGMANI_TOKEN         Bearer authentication token for non-interactive CI/CD
  NAAGMANI_API_KEY       API Key for non-interactive authentication
  NAAGMANI_ORG_ID        Default Organization ID
  NAAGMANI_PROJECT_ID    Default Project ID
  NAAGMANI_ENVIRONMENT   Default Target Environment (e.g. test, production)
`, Version)
}

func printPluginHelp() {
	fmt.Println(`Usage: naagmani plugin <subcommand> [flags]

Subcommands:
  create <name>   Create a new plugin project (--language go|node|python)
  validate [dir]  Validate plugin.json manifest against protocol rules
  dev [dir]       Build, pre-flight check, and link plugin for local OS testing
  build [dir]     Compile plugin for target platform and calculate SHA-256
  package [dir]   Create distributable .tar.gz archive and checksum
  publish [dir]   Build, package, and publish plugin version to Naagmani Cloud
  install <name>  Install or set desired plugin version for environment
  list / versions List published versions for a plugin
  update <name>   Update desired version of an installed plugin
  rollback <name> Roll back an installed plugin to prior version
  info <name>     Inspect plugin metadata, capabilities, and dependencies
  health <name>   Inspect runtime health status and diagnostics
  metrics <name>  Inspect invocation counters and execution telemetry`)
}

func printMarketplaceHelp() {
	fmt.Println(`Usage: naagmani marketplace <subcommand> [flags]

Subcommands:
  search [query]   Search the Naagmani Cloud plugin catalog (--category, --cloud-url)
  info <name>      Inspect marketplace plugin details and capabilities
  install <name>   Install a plugin from the marketplace into active environment`)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(ExitSuccess)
	}

	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Printf("naagmani CLI v%s\n", Version)
		os.Exit(ExitSuccess)

	case "help", "-h", "--help":
		printHelp()
		os.Exit(ExitSuccess)

	case "doctor":
		if err := cmd.RunDoctor(Version); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "whoami":
		if err := cmd.RunWhoami(); err != nil {
			exitWithError(ExitAuthError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "logout":
		if err := cmd.RunLogout(); err != nil {
			exitWithError(ExitInvalidArg, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "init":
		initFlags := flag.NewFlagSet("init", flag.ExitOnError)
		env := initFlags.String("env", "test", "Target environment")
		force := initFlags.Bool("force", false, "Overwrite existing naagmani.yaml")
		_ = initFlags.Parse(os.Args[2:])

		projectName := ""
		if len(initFlags.Args()) > 0 {
			projectName = initFlags.Args()[0]
		}

		if err := cmd.RunInit(cmd.InitOptions{
			ProjectName: projectName,
			Environment: *env,
			Force:       *force,
		}); err != nil {
			exitWithError(ExitInvalidArg, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "config":
		if err := cmd.RunConfig(os.Args[2:]); err != nil {
			exitWithError(ExitInvalidArg, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "login":
		loginFlags := flag.NewFlagSet("login", flag.ExitOnError)
		cloudURL := loginFlags.String("cloud-url", "", "Naagmani Cloud URL")
		email := loginFlags.String("email", "", "User email")
		password := loginFlags.String("password", "", "User password")
		_ = loginFlags.Parse(os.Args[2:])

		err := cmd.RunLogin(cmd.LoginOptions{
			CloudURL: *cloudURL,
			Email:    *email,
			Password: *password,
		})
		if err != nil {
			exitWithError(ExitAuthError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "marketplace":
		if len(os.Args) < 3 || os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help" {
			printMarketplaceHelp()
			os.Exit(ExitSuccess)
		}
		subcmd := os.Args[2]
		switch subcmd {
		case "search":
			searchFlags := flag.NewFlagSet("marketplace search", flag.ExitOnError)
			category := searchFlags.String("category", "", "Filter by category")
			cloudURL := searchFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := searchFlags.String("token", "", "Auth token")
			orgID := searchFlags.String("org-id", "", "Organization ID")
			jsonOut := searchFlags.Bool("json", false, "Output in JSON format")
			_ = searchFlags.Parse(os.Args[3:])

			query := ""
			if len(searchFlags.Args()) > 0 {
				query = searchFlags.Args()[0]
			}
			if err := cmd.RunMarketplaceSearch(cmd.MarketplaceSearchOptions{
				Query:    query,
				Category: *category,
				CloudURL: *cloudURL,
				Token:    *token,
				OrgID:    *orgID,
				JSON:     *jsonOut,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "info":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or ID required\nUsage: naagmani marketplace info <name>")
			}
			infoFlags := flag.NewFlagSet("marketplace info", flag.ExitOnError)
			cloudURL := infoFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := infoFlags.String("token", "", "Auth token")
			orgID := infoFlags.String("org-id", "", "Organization ID")
			jsonOut := infoFlags.Bool("json", false, "Output in JSON format")
			_ = infoFlags.Parse(os.Args[4:])

			if err := cmd.RunMarketplaceInfo(cmd.MarketplaceInfoOptions{
				PluginIDOrName: os.Args[3],
				CloudURL:       *cloudURL,
				Token:          *token,
				OrgID:          *orgID,
				JSON:           *jsonOut,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "versions":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or ID required\nUsage: naagmani marketplace versions <name>")
			}
			verFlags := flag.NewFlagSet("marketplace versions", flag.ExitOnError)
			cloudURL := verFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := verFlags.String("token", "", "Auth token")
			orgID := verFlags.String("org-id", "", "Organization ID")
			jsonOut := verFlags.Bool("json", false, "Output in JSON format")
			_ = verFlags.Parse(os.Args[4:])

			if err := cmd.RunMarketplaceVersions(cmd.MarketplaceVersionsOptions{
				PluginIDOrName: os.Args[3],
				CloudURL:       *cloudURL,
				Token:          *token,
				OrgID:          *orgID,
				JSON:           *jsonOut,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "install":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or ID required\nUsage: naagmani marketplace install <name>")
			}
			instFlags := flag.NewFlagSet("marketplace install", flag.ExitOnError)
			version := instFlags.String("version", "", "Target version (default: latest)")
			envID := instFlags.String("env", "", "Target environment ID")
			projID := instFlags.String("project", "", "Target project ID")
			cloudURL := instFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := instFlags.String("token", "", "Auth token")
			orgID := instFlags.String("org-id", "", "Organization ID")
			jsonOut := instFlags.Bool("json", false, "Output in JSON format")
			_ = instFlags.Parse(os.Args[4:])

			if err := cmd.RunMarketplaceInstall(cmd.MarketplaceInstallOptions{
				PluginIDOrName: os.Args[3],
				Version:        *version,
				EnvironmentID:  *envID,
				ProjectID:      *projID,
				CloudURL:       *cloudURL,
				Token:          *token,
				OrgID:          *orgID,
				JSON:           *jsonOut,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		default:
			exitWithError(ExitInvalidArg, "unknown marketplace subcommand %q (supported: search, info, versions, install)", subcmd)
		}
		os.Exit(ExitSuccess)

	case "plugin":
		if len(os.Args) < 3 || os.Args[2] == "--help" || os.Args[2] == "-h" || os.Args[2] == "help" {
			printPluginHelp()
			os.Exit(ExitSuccess)
		}

		subcmd := os.Args[2]
		switch subcmd {
		case "create":
			createFlags := flag.NewFlagSet("plugin create", flag.ExitOnError)
			lang := createFlags.String("language", "go", "Programming language (supported: go, node, python)")
			dest := createFlags.String("dest", "", "Destination directory (default: plugin name)")
			_ = createFlags.Parse(os.Args[3:])

			args := createFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "plugin name is required\nUsage: naagmani plugin create <name> [--language go|node|python]")
			}

			name := args[0]
			err := cmd.RunPluginCreate(cmd.PluginCreateOptions{
				Name:     name,
				Language: *lang,
				DestDir:  *dest,
			})
			if err != nil {
				exitWithError(ExitInvalidArg, "creating plugin: %v", err)
			}

		case "validate":
			valFlags := flag.NewFlagSet("plugin validate", flag.ExitOnError)
			_ = valFlags.Parse(os.Args[3:])
			dir := "."
			if len(valFlags.Args()) > 0 {
				dir = valFlags.Args()[0]
			}

			if err := cmd.RunPluginValidate(dir); err != nil {
				os.Exit(ExitValError)
			}

		case "dev":
			devFlags := flag.NewFlagSet("plugin dev", flag.ExitOnError)
			_ = devFlags.Parse(os.Args[3:])
			dir := "."
			if len(devFlags.Args()) > 0 {
				dir = devFlags.Args()[0]
			}

			if err := cmd.RunPluginDev(dir); err != nil {
				exitWithError(ExitValError, "running plugin dev: %v", err)
			}

		case "build":
			buildFlags := flag.NewFlagSet("plugin build", flag.ExitOnError)
			targetOS := buildFlags.String("os", "linux", "Target operating system (linux, windows, darwin)")
			targetArch := buildFlags.String("arch", "amd64", "Target architecture (amd64, arm64)")
			outDir := buildFlags.String("out", "dist", "Output directory")
			_ = buildFlags.Parse(os.Args[3:])
			dir := "."
			if len(buildFlags.Args()) > 0 {
				dir = buildFlags.Args()[0]
			}

			opts := cmd.BuildOptions{
				Dir:        dir,
				TargetOS:   *targetOS,
				TargetArch: *targetArch,
				OutputDir:  *outDir,
			}
			if _, err := cmd.RunPluginBuild(opts); err != nil {
				exitWithError(ExitValError, "building plugin: %v", err)
			}

		case "package":
			pkgFlags := flag.NewFlagSet("plugin package", flag.ExitOnError)
			targetOS := pkgFlags.String("os", "linux", "Target operating system (linux, windows, darwin)")
			targetArch := pkgFlags.String("arch", "amd64", "Target architecture (amd64, arm64)")
			outDir := pkgFlags.String("out", "dist", "Output directory")
			_ = pkgFlags.Parse(os.Args[3:])
			dir := "."
			if len(pkgFlags.Args()) > 0 {
				dir = pkgFlags.Args()[0]
			}

			opts := cmd.PackageOptions{
				Dir:        dir,
				TargetOS:   *targetOS,
				TargetArch: *targetArch,
				OutputDir:  *outDir,
			}
			if _, err := cmd.RunPluginPackage(opts); err != nil {
				exitWithError(ExitValError, "packaging plugin: %v", err)
			}

		case "publish":
			pubFlags := flag.NewFlagSet("plugin publish", flag.ExitOnError)
			cloudURL := pubFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := pubFlags.String("token", "", "Naagmani Cloud auth token")
			orgID := pubFlags.String("org-id", "", "Publisher organization ID")
			targetOS := pubFlags.String("os", "linux", "Target operating system (linux, windows, darwin)")
			targetArch := pubFlags.String("arch", "amd64", "Target architecture (amd64, arm64)")
			releaseNotes := pubFlags.String("notes", "", "Version release notes")
			_ = pubFlags.Parse(os.Args[3:])
			dir := "."
			if len(pubFlags.Args()) > 0 {
				dir = pubFlags.Args()[0]
			}

			opts := cmd.PublishOptions{
				Dir:          dir,
				CloudURL:     *cloudURL,
				Token:        *token,
				OrgID:        *orgID,
				TargetOS:     *targetOS,
				TargetArch:   *targetArch,
				ReleaseNotes: *releaseNotes,
			}
			if _, err := cmd.RunPluginPublish(opts); err != nil {
				exitWithError(ExitNetError, "publishing plugin: %v", err)
			}

		case "versions", "list":
			verFlags := flag.NewFlagSet("plugin versions", flag.ExitOnError)
			cloudURL := verFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := verFlags.String("token", "", "Naagmani Cloud auth token")
			orgID := verFlags.String("org-id", "", "Organization ID")
			_ = verFlags.Parse(os.Args[3:])

			if len(verFlags.Args()) < 1 {
				exitWithError(ExitInvalidArg, "plugin name is required\nUsage: naagmani plugin versions <name>")
			}

			pluginName := verFlags.Args()[0]
			opts := cmd.VersionsOptions{
				PluginName: pluginName,
				CloudURL:   *cloudURL,
				Token:      *token,
				OrgID:      *orgID,
			}
			if err := cmd.RunPluginVersions(opts); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "update", "install":
			updateFlags := flag.NewFlagSet("plugin update", flag.ExitOnError)
			targetVer := updateFlags.String("version", "", "Target version to update to")
			perms := updateFlags.String("approved-permissions", "", "Comma-separated approved permissions")
			cloudURL := updateFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := updateFlags.String("token", "", "Naagmani Cloud auth token")
			orgID := updateFlags.String("org-id", "", "Organization ID")
			_ = updateFlags.Parse(os.Args[3:])

			if len(updateFlags.Args()) < 1 {
				exitWithError(ExitInvalidArg, "plugin name is required\nUsage: naagmani plugin update <name> --version <version>")
			}

			pluginName := updateFlags.Args()[0]
			var approvedPerms []string
			if *perms != "" {
				for _, p := range strings.Split(*perms, ",") {
					trimmed := strings.TrimSpace(p)
					if trimmed != "" {
						approvedPerms = append(approvedPerms, trimmed)
					}
				}
			}

			opts := cmd.UpdateOptions{
				PluginName:          pluginName,
				TargetVersion:       *targetVer,
				ApprovedPermissions: approvedPerms,
				CloudURL:            *cloudURL,
				Token:               *token,
				OrgID:               *orgID,
			}
			if err := cmd.RunPluginUpdate(opts); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "rollback":
			rbFlags := flag.NewFlagSet("plugin rollback", flag.ExitOnError)
			targetVer := rbFlags.String("version", "", "Optional target version to rollback to")
			cloudURL := rbFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := rbFlags.String("token", "", "Naagmani Cloud auth token")
			orgID := rbFlags.String("org-id", "", "Organization ID")
			_ = rbFlags.Parse(os.Args[3:])

			if len(rbFlags.Args()) < 1 {
				exitWithError(ExitInvalidArg, "plugin name is required\nUsage: naagmani plugin rollback <name>")
			}

			pluginName := rbFlags.Args()[0]
			opts := cmd.RollbackOptions{
				PluginName:    pluginName,
				TargetVersion: *targetVer,
				CloudURL:      *cloudURL,
				Token:         *token,
				OrgID:         *orgID,
			}
			if err := cmd.RunPluginRollback(opts); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "info":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or path required\nUsage: naagmani plugin info <name>")
			}
			if err := cmd.RunPluginInfo(os.Args[3]); err != nil {
				exitWithError(ExitInvalidArg, "%v", err)
			}

		case "capabilities":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or path required\nUsage: naagmani plugin capabilities <name>")
			}
			if err := cmd.RunPluginCapabilities(os.Args[3]); err != nil {
				exitWithError(ExitInvalidArg, "%v", err)
			}

		case "dependencies":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or path required\nUsage: naagmani plugin dependencies <name>")
			}
			if err := cmd.RunPluginDependencies(os.Args[3]); err != nil {
				exitWithError(ExitInvalidArg, "%v", err)
			}

		case "health":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or path required\nUsage: naagmani plugin health <name>")
			}
			if err := cmd.RunPluginHealth(os.Args[3]); err != nil {
				exitWithError(ExitInvalidArg, "%v", err)
			}

		case "metrics":
			if len(os.Args) < 4 {
				exitWithError(ExitInvalidArg, "plugin name or path required\nUsage: naagmani plugin metrics <name>")
			}
			if err := cmd.RunPluginMetrics(os.Args[3]); err != nil {
				exitWithError(ExitInvalidArg, "%v", err)
			}

		default:
			exitWithError(ExitInvalidArg, "unknown plugin subcommand %q\nRun 'naagmani help' for usage.", subcmd)
		}
		os.Exit(ExitSuccess)

	case "policy":
		if len(os.Args) < 3 {
			exitWithError(ExitInvalidArg, "missing policy subcommand (list, get, create, update, enable, disable, rollback, simulate)")
		}

		subcmd := os.Args[2]
		switch subcmd {
		case "list":
			listFlags := flag.NewFlagSet("policy list", flag.ExitOnError)
			cloudURL := listFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := listFlags.String("token", "", "Auth token")
			orgID := listFlags.String("org", "", "Organization ID")
			scope := listFlags.String("scope", "", "Filter by scope")
			status := listFlags.String("status", "", "Filter by status")
			_ = listFlags.Parse(os.Args[3:])

			if err := cmd.RunPolicyList(cmd.PolicyListOptions{
				CloudURL: *cloudURL,
				Token:    *token,
				OrgID:    *orgID,
				Scope:    *scope,
				Status:   *status,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "get":
			getFlags := flag.NewFlagSet("policy get", flag.ExitOnError)
			cloudURL := getFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := getFlags.String("token", "", "Auth token")
			orgID := getFlags.String("org", "", "Organization ID")
			_ = getFlags.Parse(os.Args[3:])

			args := getFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "policy ID required\nUsage: naagmani policy get <id>")
			}
			if err := cmd.RunPolicyGet(cmd.PolicyGetOptions{
				PolicyID: args[0],
				CloudURL: *cloudURL,
				Token:    *token,
				OrgID:    *orgID,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "create":
			createFlags := flag.NewFlagSet("policy create", flag.ExitOnError)
			name := createFlags.String("name", "", "Policy name")
			desc := createFlags.String("description", "", "Policy description")
			scope := createFlags.String("scope", "organization", "Policy scope (organization, project, environment, principal)")
			priority := createFlags.Int("priority", 0, "Policy priority integer")
			effect := createFlags.String("effect", "ALLOW", "Policy effect (ALLOW or DENY)")
			models := createFlags.String("models", "", "Comma-separated model names")
			providers := createFlags.String("providers", "", "Comma-separated provider names")
			plugins := createFlags.String("plugins", "", "Comma-separated plugin names")
			capabilities := createFlags.String("capabilities", "", "Comma-separated capability names")
			tools := createFlags.String("tools", "", "Comma-separated tool names")
			environments := createFlags.String("environments", "", "Comma-separated environment names")
			principals := createFlags.String("principals", "", "Comma-separated principal IDs")
			maxSteps := createFlags.Int("max-steps", 0, "Max agent steps")
			maxToolCalls := createFlags.Int("max-tool-calls", 0, "Max tool calls")
			cloudURL := createFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := createFlags.String("token", "", "Auth token")
			orgID := createFlags.String("org", "", "Organization ID")
			_ = createFlags.Parse(os.Args[3:])

			if err := cmd.RunPolicyCreate(cmd.PolicyCreateOptions{
				CloudURL:     *cloudURL,
				Token:        *token,
				OrgID:        *orgID,
				Name:         *name,
				Description:  *desc,
				Scope:        *scope,
				Priority:     *priority,
				Effect:       *effect,
				Models:       *models,
				Providers:    *providers,
				Plugins:      *plugins,
				Capabilities: *capabilities,
				Tools:        *tools,
				Environments: *environments,
				Principals:   *principals,
				MaxSteps:     *maxSteps,
				MaxToolCalls: *maxToolCalls,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "update":
			updateFlags := flag.NewFlagSet("policy update", flag.ExitOnError)
			name := updateFlags.String("name", "", "Policy name")
			desc := updateFlags.String("description", "", "Policy description")
			priority := updateFlags.Int("priority", 0, "Priority")
			effect := updateFlags.String("effect", "", "Effect (ALLOW/DENY)")
			cloudURL := updateFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := updateFlags.String("token", "", "Auth token")
			orgID := updateFlags.String("org", "", "Organization ID")
			_ = updateFlags.Parse(os.Args[3:])

			args := updateFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "policy ID required\nUsage: naagmani policy update <id>")
			}
			if err := cmd.RunPolicyUpdate(cmd.PolicyUpdateOptions{
				PolicyID:    args[0],
				CloudURL:    *cloudURL,
				Token:       *token,
				OrgID:       *orgID,
				Name:        *name,
				Description: *desc,
				Priority:    *priority,
				Effect:      *effect,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "enable":
			enableFlags := flag.NewFlagSet("policy enable", flag.ExitOnError)
			cloudURL := enableFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := enableFlags.String("token", "", "Auth token")
			orgID := enableFlags.String("org", "", "Organization ID")
			_ = enableFlags.Parse(os.Args[3:])

			args := enableFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "policy ID required\nUsage: naagmani policy enable <id>")
			}
			if err := cmd.RunPolicyEnable(cmd.PolicyActionOptions{
				PolicyID: args[0],
				CloudURL: *cloudURL,
				Token:    *token,
				OrgID:    *orgID,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "disable":
			disableFlags := flag.NewFlagSet("policy disable", flag.ExitOnError)
			cloudURL := disableFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := disableFlags.String("token", "", "Auth token")
			orgID := disableFlags.String("org", "", "Organization ID")
			_ = disableFlags.Parse(os.Args[3:])

			args := disableFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "policy ID required\nUsage: naagmani policy disable <id>")
			}
			if err := cmd.RunPolicyDisable(cmd.PolicyActionOptions{
				PolicyID: args[0],
				CloudURL: *cloudURL,
				Token:    *token,
				OrgID:    *orgID,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "rollback":
			rbFlags := flag.NewFlagSet("policy rollback", flag.ExitOnError)
			targetVer := rbFlags.Int("target-version", 0, "Target version to roll back to")
			cloudURL := rbFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := rbFlags.String("token", "", "Auth token")
			orgID := rbFlags.String("org", "", "Organization ID")
			_ = rbFlags.Parse(os.Args[3:])

			args := rbFlags.Args()
			if len(args) < 1 {
				exitWithError(ExitInvalidArg, "policy ID required\nUsage: naagmani policy rollback <id>")
			}
			if err := cmd.RunPolicyRollback(cmd.PolicyActionOptions{
				PolicyID:      args[0],
				CloudURL:      *cloudURL,
				Token:         *token,
				OrgID:         *orgID,
				TargetVersion: *targetVer,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		case "simulate":
			simFlags := flag.NewFlagSet("policy simulate", flag.ExitOnError)
			env := simFlags.String("environment", "", "Environment ID")
			principal := simFlags.String("principal", "", "Principal ID")
			model := simFlags.String("model", "", "Model name")
			prov := simFlags.String("provider", "", "Provider name")
			plug := simFlags.String("plugin", "", "Plugin name")
			cap := simFlags.String("capability", "", "Capability name")
			tool := simFlags.String("tool", "", "Tool name")
			reqType := simFlags.String("request-type", "", "Request type")
			cloudURL := simFlags.String("cloud-url", "", "Naagmani Cloud URL")
			token := simFlags.String("token", "", "Auth token")
			orgID := simFlags.String("org", "", "Organization ID")
			_ = simFlags.Parse(os.Args[3:])

			if err := cmd.RunPolicySimulate(cmd.PolicySimulateOptions{
				CloudURL:      *cloudURL,
				Token:         *token,
				OrgID:         *orgID,
				EnvironmentID: *env,
				PrincipalID:   *principal,
				Model:         *model,
				Provider:      *prov,
				Plugin:        *plug,
				Capability:    *cap,
				Tool:          *tool,
				RequestType:   *reqType,
			}); err != nil {
				exitWithError(ExitNetError, "%v", err)
			}

		default:
			exitWithError(ExitInvalidArg, "unknown policy subcommand: %q", subcmd)
		}
		os.Exit(ExitSuccess)

	case "usage":
		if err := cmd.ExecuteUsage(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "cost":
		if err := cmd.ExecuteCost(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "budget":
		if err := cmd.ExecuteBudget(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "quota":
		if err := cmd.ExecuteQuota(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "platform":
		if err := cmd.RunPlatform(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	case "fleet":
		if err := cmd.RunFleet(os.Args[2:]); err != nil {
			exitWithError(ExitNetError, "%v", err)
		}
		os.Exit(ExitSuccess)

	default:
		exitWithError(ExitInvalidArg, "unknown command: %q\nRun 'naagmani help' for usage.", os.Args[1])
	}
}
