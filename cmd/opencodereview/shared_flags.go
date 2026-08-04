package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func addRepoFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "repo", "", "root directory of the git repository (default: current dir)")
}

func addRuleFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "rule", "", "path to JSON file with system review rules")
}

func addDiffFlags(cmd *cobra.Command, from, to, commit *string) {
	cmd.Flags().StringVar(from, "from", "", "source ref to start diff from (e.g., 'main')")
	cmd.Flags().StringVar(to, "to", "", "target ref to end diff at (e.g., 'feature-branch')")
	cmd.Flags().StringVarP(commit, "commit", "c", "", "single commit hash or tag to review (vs its parent)")
}

func addBackgroundFlags(cmd *cobra.Command, background, backgroundFile *string) {
	cmd.Flags().StringVarP(background, "background", "b", "", "optional requirement/business context for the review")
	cmd.Flags().StringVarP(backgroundFile, "background-file", "B", "", "path to a Markdown file used as review background")
}

func addOutputFlags(cmd *cobra.Command, format, audience *string) {
	cmd.Flags().StringVarP(format, "format", "f", "text", "output format: text or json")
	cmd.Flags().StringVar(audience, "audience", "human", "output audience: human (show progress) or agent (summary only)")
	cmd.RegisterFlagCompletionFunc("format", completeEnum("text", "json"))
	cmd.RegisterFlagCompletionFunc("audience", completeEnum("human", "agent"))
}

func addExcludeFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "exclude", "", "comma-separated gitignore-style patterns to exclude; merged with rule.json excludes")
}

func addRunnerFlags(cmd *cobra.Command, runnerName, runnerModel *string, runnerTimeout *int) {
	cmd.Flags().StringVar(runnerName, "runner", "", "local subscription runner to use: codex or claude")
	cmd.Flags().StringVar(runnerModel, "runner-model", "", "optional model passed to the selected local runner")
	cmd.Flags().IntVar(runnerTimeout, "timeout", 10, "local runner process timeout in minutes")
	cmd.RegisterFlagCompletionFunc("runner", completeEnum("codex", "claude"))
}

func addPreviewFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVarP(target, "preview", "p", false, "preview which files will be reviewed without running a local runner")
}

func completeEnum(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

// --- Validation functions ---

func validateDiffMode(from, to, commit string) error {
	modeCount := 0
	if from != "" || to != "" {
		modeCount++
	}
	if commit != "" {
		modeCount++
	}
	if modeCount > 1 {
		return fmt.Errorf("only one review mode allowed (--from/--to or --commit)")
	}
	if from != "" && to == "" {
		return fmt.Errorf("--to is required when --from is specified")
	}
	if to != "" && from == "" {
		return fmt.Errorf("--from is required when --to is specified")
	}
	return nil
}

func validateAudience(audience string) error {
	switch audience {
	case "human", "agent":
		return nil
	default:
		return fmt.Errorf("invalid --audience value %q: must be 'human' or 'agent'", audience)
	}
}

func validateReviewOptions(opts *reviewOptions) error {
	if err := validateDiffMode(opts.from, opts.to, opts.commit); err != nil {
		return err
	}
	if opts.preview && opts.resume != "" {
		return fmt.Errorf("--preview and --resume cannot be used together")
	}
	if err := validateAudience(opts.audience); err != nil {
		return err
	}
	if opts.maxGitProcs < 0 {
		return fmt.Errorf("--max-git-procs must be a non-negative integer (0 means use default 16)")
	}
	if opts.runnerTimeout <= 0 {
		return fmt.Errorf("--timeout must be a positive integer number of minutes")
	}
	if !opts.preview {
		if opts.runner == "" {
			return fmt.Errorf("--runner is required for review (use --preview to inspect files without invoking a runner)")
		}
		if err := validateRunner(opts.runner); err != nil {
			return err
		}
	} else if opts.runner != "" {
		if err := validateRunner(opts.runner); err != nil {
			return err
		}
	}
	return nil
}

func validateScanOptions(opts *scanOptions) error {
	if err := validateAudience(opts.audience); err != nil {
		return err
	}
	if opts.maxGitProcs < 0 {
		return fmt.Errorf("--max-git-procs must be a non-negative integer (0 means use default 16)")
	}
	if opts.runnerTimeout <= 0 {
		return fmt.Errorf("--timeout must be a positive integer number of minutes")
	}
	if opts.preview && opts.resume != "" {
		return fmt.Errorf("--preview and --resume cannot be used together")
	}
	if !opts.preview {
		if opts.runner == "" {
			return fmt.Errorf("--runner is required for scan (use --preview to inspect files without invoking a runner)")
		}
		if err := validateRunner(opts.runner); err != nil {
			return err
		}
	} else if opts.runner != "" {
		if err := validateRunner(opts.runner); err != nil {
			return err
		}
	}
	return nil
}

func validateRunner(kind string) error {
	switch kind {
	case "codex", "claude":
		return nil
	default:
		return fmt.Errorf("invalid --runner value %q: must be 'codex' or 'claude'", kind)
	}
}

func validateDelegateOptions(opts *delegateOptions) error {
	return validateDiffMode(opts.from, opts.to, opts.commit)
}

// registerReviewFlags registers all review command flags on cmd, binding to opts.
func registerReviewFlags(cmd *cobra.Command, opts *reviewOptions) {
	addRuleFlag(cmd, &opts.rulePath)
	addRepoFlag(cmd, &opts.repoDir)
	addDiffFlags(cmd, &opts.from, &opts.to, &opts.commit)
	cmd.Flags().StringVar(&opts.resume, "resume", "", "resume from a previous review session id")
	cmd.RegisterFlagCompletionFunc("resume", completeSessionIDs)
	addExcludeFlag(cmd, &opts.excludes)
	addOutputFlags(cmd, &opts.outputFormat, &opts.audience)
	cmd.Flags().IntVar(&opts.maxGitProcs, "max-git-procs", 16, "max concurrent git subprocesses")
	addBackgroundFlags(cmd, &opts.background, &opts.backgroundFile)
	addRunnerFlags(cmd, &opts.runner, &opts.runnerModel, &opts.runnerTimeout)
	addPreviewFlag(cmd, &opts.preview)
}

// registerScanFlags registers all scan command flags on cmd, binding to opts.
func registerScanFlags(cmd *cobra.Command, opts *scanOptions) {
	addRuleFlag(cmd, &opts.rulePath)
	addRepoFlag(cmd, &opts.repoDir)
	cmd.Flags().StringVar(&opts.paths, "path", "", "comma-separated repo-relative directories or files to scan (default: whole repo)")
	addExcludeFlag(cmd, &opts.excludes)
	addOutputFlags(cmd, &opts.outputFormat, &opts.audience)
	cmd.Flags().IntVar(&opts.maxGitProcs, "max-git-procs", 16, "max concurrent git subprocesses")
	cmd.Flags().StringVarP(&opts.background, "background", "b", "", "optional requirement/business context for the scan")
	cmd.Flags().BoolVarP(&opts.preview, "preview", "p", false, "preview which files will be scanned without running the LLM")
	cmd.Flags().BoolVar(&opts.noPlan, "no-plan", false, "skip the per-file PLAN_TASK pre-pass")
	cmd.Flags().BoolVar(&opts.noDedup, "no-dedup", false, "skip the per-batch DEDUP_TASK")
	cmd.Flags().BoolVar(&opts.noSummary, "no-summary", false, "skip the post-run PROJECT_SUMMARY_TASK")
	cmd.Flags().StringVar(&opts.batch, "batch", "", "override BATCH_STRATEGY: none | by-language | by-directory")
	addRunnerFlags(cmd, &opts.runner, &opts.runnerModel, &opts.runnerTimeout)
	cmd.Flags().StringVar(&opts.resume, "resume", "", "resume from a previous scan session id")
	cmd.RegisterFlagCompletionFunc("batch", completeEnum("none", "by-language", "by-directory"))
}

// registerDelegateFlags registers all delegate shared flags on cmd, binding to opts.
func registerDelegateFlags(cmd *cobra.Command, opts *delegateOptions) {
	addRepoFlag(cmd, &opts.repoDir)
	addDiffFlags(cmd, &opts.from, &opts.to, &opts.commit)
	addExcludeFlag(cmd, &opts.excludes)
	addRuleFlag(cmd, &opts.rulePath)
	addBackgroundFlags(cmd, &opts.background, &opts.backgroundFile)
	cmd.Flags().IntVar(&opts.maxGitProcs, "max-git-procs", 16, "max concurrent git subprocesses")
}
