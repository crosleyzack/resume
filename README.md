# Description

Resume template developed by Zack Crosley. Template available for use under MIT License.
The actual resume document is `resume.pdf`

# Building

Run `compile-latex.yaml` workflow to build the pdf from latex files.

`LaTeX Workshop: Build LaTeX project` can be run in vscode using the devcontainer.

Finally, you can use nix via `nix-shell -p texliveFull --run "pdflatex resume.tex"`
