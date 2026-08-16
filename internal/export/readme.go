package export

import (
	"fmt"
	"strings"
)

// readme builds the README.txt that ships inside the export. It is written in
// English to match the rest of the command-line output, and it is the only
// place the recipient can learn how to compile the project, so it states the
// command chain, the packages, and the fact that this copy is a dead end.
func readme(mode BibMode) string {
	var b strings.Builder

	b.WriteString("NodePaper LaTeX export\n")
	b.WriteString("======================\n\n")

	b.WriteString("This directory is a standalone LaTeX project generated from a NodePaper\n")
	b.WriteString("(Markdown) project. It compiles on its own; NodePaper is not needed to build\n")
	b.WriteString("it, and nothing here refers back to the original project.\n\n")

	b.WriteString("Contents\n")
	b.WriteString("--------\n")
	b.WriteString("  paper.tex       the document\n")
	if mode.needsBibFile() {
		b.WriteString("  references.bib  the bibliography database\n")
	}
	b.WriteString("  images/         only the images the document actually references\n")
	b.WriteString("  *.tex elsewhere any LaTeX fragments the document \\input{}s\n")
	b.WriteString("  README.txt      this file\n\n")

	b.WriteString("How to compile\n")
	b.WriteString("--------------\n")
	b.WriteString(fmt.Sprintf("Bibliography mode: %s\n", bibModeDescription(mode)))
	b.WriteString("Run these commands in this directory, in this order:\n\n")
	for _, step := range compileChain(mode) {
		b.WriteString(fmt.Sprintf("  %s %s\n", step.tool, strings.Join(step.args, " ")))
	}
	b.WriteString("\n")
	b.WriteString("The engine must be XeLaTeX. pdflatex and lualatex will not work: the document\n")
	b.WriteString("selects Chinese fonts through ctex/fontspec, which requires XeLaTeX or LuaLaTeX,\n")
	b.WriteString("and the layout was set for XeLaTeX.\n\n")
	if len(compileChain(mode)) > 2 {
		b.WriteString("The repeated XeLaTeX runs are not redundant: the first pass writes the\n")
		b.WriteString("citation and cross-reference data, and the later passes read it back.\n\n")
	}

	b.WriteString("Required LaTeX packages\n")
	b.WriteString("-----------------------\n")
	b.WriteString("A full TeX Live installation already has everything. A minimal MiKTeX or\n")
	b.WriteString("TeX Live installation needs these:\n\n")
	b.WriteString("  ctex fontspec xecjk geometry amsmath amsfonts graphicx longtable booktabs\n")
	b.WriteString("  array multirow tabularx float fvextra mdframed xcolor url xurl hyperref\n")
	b.WriteString("  cleveref caption newtx\n")
	switch mode {
	case BibBibTeX:
		b.WriteString("  gbt7714      (the GB/T 7714-2015 BibTeX style, pulls in natbib)\n")
	case BibBibLaTeX:
		b.WriteString("  biblatex biblatex-gb7714-2015 biber\n")
		b.WriteString("             (biber is a separate program, not a LaTeX package; MiKTeX\n")
		b.WriteString("             installations often do not have it until it is installed)\n")
	}
	b.WriteString("\nInstall commands:\n\n")
	b.WriteString(fmt.Sprintf("  TeX Live   tlmgr install %s\n", strings.Join(packageList(mode), " ")))
	b.WriteString(fmt.Sprintf("  MiKTeX     miktex packages install %s\n", strings.Join(packageList(mode), " ")))
	b.WriteString("\n")
	b.WriteString("Chinese text needs Chinese fonts. On Windows the document expects SimSun,\n")
	b.WriteString("SimHei and KaiTi; SimHei and KaiTi come from the optional Windows feature\n")
	b.WriteString("\"Chinese (Simplified) Supplemental Fonts\". Without them XeLaTeX still\n")
	b.WriteString("compiles, but bold and italic Chinese are synthesised and look softer.\n\n")

	if mode == BibBibTeX {
		b.WriteString("Note on English titles (BibTeX mode)\n")
		b.WriteString("------------------------------------\n")
		b.WriteString("The gbt7714 style rewrites English titles to sentence case, so\n")
		b.WriteString("\"Demand Forecasting for Shared Mobility Systems\" is typeset as\n")
		b.WriteString("\"Demand forecasting for shared mobility systems\". To keep a title exactly\n")
		b.WriteString("as written, wrap it in a second pair of braces in references.bib:\n\n")
		b.WriteString("  title = {{Demand Forecasting for Shared Mobility Systems}},\n\n")
		b.WriteString("The extra braces tell BibTeX the capitalisation is significant. This applies\n")
		b.WriteString("per entry, so only the titles you brace are protected.\n\n")
	}

	b.WriteString("This export is one-way\n")
	b.WriteString("----------------------\n")
	b.WriteString("These files were generated from the Markdown project and are never read back\n")
	b.WriteString("into it. Anything you change here - text, formatting, references, images -\n")
	b.WriteString("stays here. Re-running `nodepaper export` regenerates paper.tex from the\n")
	b.WriteString("Markdown and overwrites your edits.\n\n")
	b.WriteString("So pick one side and stay on it: either keep editing the Markdown project and\n")
	b.WriteString("treat this directory as disposable output, or take this directory as the new\n")
	b.WriteString("source and stop exporting into it.\n\n")

	b.WriteString("Version control\n")
	b.WriteString("---------------\n")
	b.WriteString("No .gitignore was created; that choice belongs to whoever owns the repository.\n")
	b.WriteString("If you put this directory under version control, the files worth committing are\n")
	b.WriteString("paper.tex, ")
	if mode.needsBibFile() {
		b.WriteString("references.bib, ")
	}
	b.WriteString("the images and this README. LaTeX regenerates everything\n")
	b.WriteString("else on every run, so these are usually ignored:\n\n")
	b.WriteString("  *.aux *.log *.out *.toc *.lof *.lot *.fls *.fdb_latexmk *.synctex.gz\n")
	switch mode {
	case BibBibTeX:
		b.WriteString("  *.bbl *.blg\n")
	case BibBibLaTeX:
		b.WriteString("  *.bbl *.blg *.bcf *.run.xml\n")
	}
	b.WriteString("  paper.pdf   (or keep it, if the PDF is part of what you deliver)\n")

	return b.String()
}

func bibModeDescription(mode BibMode) string {
	switch mode {
	case BibBibTeX:
		return "bibtex - \\cite commands resolved by bibtex with the gbt7714 style"
	case BibBibLaTeX:
		return "biblatex - \\autocite commands resolved by biber with biblatex-gb7714-2015"
	default:
		return "inline - the reference list is already typeset into paper.tex; there is no .bib file and no bibliography step"
	}
}

// packageList is the set a minimal installation is most likely to be missing.
// It is kept short on purpose: a command that tries to install forty packages
// fails on the first name a distribution spells differently.
func packageList(mode BibMode) []string {
	packages := []string{"ctex", "fvextra", "mdframed", "xurl", "cleveref", "newtx"}
	switch mode {
	case BibBibTeX:
		packages = append(packages, "gbt7714", "natbib")
	case BibBibLaTeX:
		packages = append(packages, "biblatex", "biblatex-gb7714-2015", "biber")
	}
	return packages
}
