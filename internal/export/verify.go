package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nodepaper/internal/diagnostic"
)

// verifyStep is one command of the compile chain.
type verifyStep struct {
	tool string
	args []string
}

// compileChain returns the exact command sequence a recipient of this export
// would run. README.txt prints it, the terminal prints it as the next step, and
// --verify executes it, so all three are the same list and cannot drift apart.
//
// hasBibliography comes from the .tex that was actually produced, not from the
// mode: a document that cites nothing and nocites nothing gets no
// \bibliography, and then bibtex or biber would only fail with "I found no
// \citation commands" on a project whose PDF is already correct. Deciding this
// from the emitted file keeps the chain honest for a paper that maintains its
// references by hand.
func compileChain(mode BibMode, hasBibliography bool) []verifyStep {
	xelatex := verifyStep{tool: "xelatex", args: []string{"paper.tex"}}
	if !hasBibliography {
		return []verifyStep{xelatex, xelatex}
	}
	switch mode {
	case BibBibTeX:
		return []verifyStep{xelatex, {tool: "bibtex", args: []string{"paper"}}, xelatex, xelatex}
	case BibBibLaTeX:
		return []verifyStep{xelatex, {tool: "biber", args: []string{"paper"}}, xelatex, xelatex}
	default:
		return []verifyStep{xelatex, xelatex}
	}
}

// CompileCommands renders the chain as the command lines a person would type.
func CompileCommands(mode BibMode, hasBibliography bool) []string {
	steps := compileChain(mode, hasBibliography)
	commands := make([]string, 0, len(steps))
	for _, step := range steps {
		commands = append(commands, strings.TrimSpace(step.tool+" "+strings.Join(step.args, " ")))
	}
	return commands
}

// batchArgs adds the flags that keep an unattended XeLaTeX run from stopping at
// an interactive prompt. They belong to verification, not to the instructions:
// a person running the chain by hand wants the prompt.
func batchArgs(tool string, args []string) []string {
	if strings.HasPrefix(filepath.Base(tool), "xelatex") {
		return append([]string{"-interaction=nonstopmode", "-halt-on-error"}, args...)
	}
	return args
}

// verify compiles a throwaway copy of the export and reports whether the chain
// succeeded. It never writes into the delivered directory: the copy lives in a
// temporary directory that is removed before this function returns, so no
// .aux, .log, .bbl or .pdf is left behind for the recipient to wonder about.
func verify(ctx context.Context, executor commandExecutor, logger *logWriter, exportDir string, mode BibMode, hasBibliography bool) (bool, []diagnostic.Diagnostic) {
	steps := compileChain(mode, hasBibliography)

	resolved := make([]verifyStep, 0, len(steps))
	for _, step := range steps {
		path, err := lookPath(step.tool)
		if err != nil {
			return false, []diagnostic.Diagnostic{warningDiag(CodeVerifySkipped,
				fmt.Sprintf("--verify did not run: %s was not found on PATH", step.tool),
				fmt.Sprintf("The export itself is complete and unaffected. Install a TeX distribution that provides %s, then run the export again with --verify.", step.tool))}
		}
		resolved = append(resolved, verifyStep{tool: path, args: batchArgs(step.tool, step.args)})
	}

	scratch, err := os.MkdirTemp("", "nodepaper-export-verify-")
	if err != nil {
		return false, []diagnostic.Diagnostic{warningDiag(CodeVerifyWorkspace,
			fmt.Sprintf("--verify did not run: cannot create a temporary directory: %v", err),
			"The export itself is complete and unaffected.")}
	}
	defer func() {
		if removeErr := os.RemoveAll(scratch); removeErr != nil {
			logger.Printf("Verify scratch directory not removed: %v", removeErr)
		}
	}()

	if err := copyTree(exportDir, scratch); err != nil {
		return false, []diagnostic.Diagnostic{warningDiag(CodeVerifyWorkspace,
			fmt.Sprintf("--verify did not run: cannot stage the export for compilation: %v", err),
			"The export itself is complete and unaffected.")}
	}
	logger.Printf("Verify: compiling a copy of the export in %s", scratch)

	for index, step := range resolved {
		name := filepath.Base(step.tool)
		logger.Printf("Verify step %d/%d: %s %s", index+1, len(resolved), name, strings.Join(step.args, " "))
		processResult, runErr := executor.Run(ctx, scratch, step.tool, step.args...)
		logger.Printf("Verify step %d exit code: %d", index+1, processResult.ExitCode)
		if processResult.Stdout != "" {
			logger.Printf("stdout:\n%s", processResult.Stdout)
		}
		if processResult.Stderr != "" {
			logger.Printf("stderr:\n%s", processResult.Stderr)
		}
		if runErr == nil && processResult.ExitCode == 0 {
			continue
		}
		return false, []diagnostic.Diagnostic{errorDiag(CodeVerifyFailed,
			fmt.Sprintf("--verify failed at step %d of %d (%s %s): exit code %d",
				index+1, len(resolved), name, strings.Join(step.args, " "), processResult.ExitCode),
			"The exported files are on disk. Run the command chain from README.txt in the export "+
				"directory to see the full LaTeX output, and check that the packages listed there are installed.")}
	}

	logger.Printf("Verify: the compile chain succeeded")
	return true, nil
}
