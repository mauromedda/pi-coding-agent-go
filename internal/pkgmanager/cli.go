// ABOUTME: CLI dispatch for package subcommands: install, remove, update, list
// ABOUTME: Routes to source-specific installers and updates the manifest file

package pkgmanager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// RunCLI dispatches package subcommands: install, remove, update, list.
// args contains the subcommand followed by its arguments.
// The -l flag selects local scope (project-local instead of global).
func RunCLI(args []string, globalDir, localDir string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: package <install|remove|update|list> [flags] [spec...]")
	}

	subcmd := args[0]
	rest := args[1:]

	local, rest := extractFlag(rest, "-l")
	destDir := globalDir
	if local {
		destDir = localDir
	}

	installers := map[Source]Installer{
		SourceNPM:   &NPMInstaller{},
		SourceGit:   &GitInstaller{},
		SourceLocal: &LocalInstaller{},
	}

	switch subcmd {
	case "install":
		return runInstall(rest, destDir, local, installers)
	case "remove":
		return runRemove(rest, destDir, local, installers)
	case "update":
		return runUpdate(rest, destDir, local, installers)
	case "list":
		return runList(destDir, installers)
	default:
		return fmt.Errorf("unknown subcommand %q: expected install, remove, update, or list", subcmd)
	}
}

func runInstall(args []string, destDir string, local bool, installers map[Source]Installer) error {
	if len(args) == 0 {
		return fmt.Errorf("install requires at least one package spec")
	}

	ctx := context.Background()
	manifest, err := LoadManifest(destDir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	for _, raw := range args {
		spec := ParseSpec(raw)
		inst, ok := installers[spec.Source]
		if !ok {
			return fmt.Errorf("no installer for source %s", spec.Source)
		}

		info, err := inst.Install(ctx, spec, destDir)
		if err != nil {
			return fmt.Errorf("installing %s: %w", spec.Name, err)
		}
		info.Local = local
		manifest.Add(*info)

		fmt.Printf("installed %s (%s) version %s\n", info.Name, info.Source, info.Version)
	}

	return SaveManifest(destDir, manifest)
}

func runRemove(args []string, destDir string, local bool, installers map[Source]Installer) error {
	if len(args) == 0 {
		return fmt.Errorf("remove requires at least one package spec")
	}

	manifest, err := LoadManifest(destDir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	for _, raw := range args {
		spec := ParseSpec(raw)
		inst, ok := installers[spec.Source]
		if !ok {
			return fmt.Errorf("no installer for source %s", spec.Source)
		}

		if err := inst.Remove(spec, destDir); err != nil {
			return fmt.Errorf("removing %s: %w", spec.Name, err)
		}
		manifest.Remove(spec.Name, local)

		fmt.Printf("removed %s\n", spec.Name)
	}

	return SaveManifest(destDir, manifest)
}

func runUpdate(args []string, destDir string, local bool, installers map[Source]Installer) error {
	if len(args) == 0 {
		return fmt.Errorf("update requires at least one package spec")
	}

	ctx := context.Background()
	manifest, err := LoadManifest(destDir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	for _, raw := range args {
		spec := ParseSpec(raw)
		inst, ok := installers[spec.Source]
		if !ok {
			return fmt.Errorf("no installer for source %s", spec.Source)
		}

		info, err := inst.Update(ctx, spec, destDir)
		if err != nil {
			return fmt.Errorf("updating %s: %w", spec.Name, err)
		}
		info.Local = local
		manifest.Add(*info)

		fmt.Printf("updated %s (%s) version %s\n", info.Name, info.Source, info.Version)
	}

	return SaveManifest(destDir, manifest)
}

func runList(destDir string, installers map[Source]Installer) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tVERSION\tPATH")

	for _, source := range []Source{SourceNPM, SourceGit, SourceLocal} {
		inst := installers[source]
		infos, err := inst.List(destDir)
		if err != nil {
			return fmt.Errorf("listing %s packages: %w", source, err)
		}
		for _, info := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, info.Source, info.Version, info.Path)
		}
	}

	return w.Flush()
}

// extractFlag removes a flag from args and returns whether it was present.
func extractFlag(args []string, flag string) (bool, []string) {
	var filtered []string
	found := false
	for _, a := range args {
		if a == flag {
			found = true
			continue
		}
		filtered = append(filtered, a)
	}
	return found, filtered
}

// FormatSource converts a source string back to Source type.
func FormatSource(s string) (Source, error) {
	switch strings.ToLower(s) {
	case "npm":
		return SourceNPM, nil
	case "git":
		return SourceGit, nil
	case "local":
		return SourceLocal, nil
	default:
		return 0, fmt.Errorf("unknown source %q", s)
	}
}
