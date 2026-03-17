## Description

Resume template developed by Zack Crosley. Template available for use under MIT License.
The actual resume document is `resume.pdf`

## Building

Run `compile-latex.yaml` workflow to build the pdf from latex files.

`LaTeX Workshop: Build LaTeX project` can be run in vscode using the devcontainer.

Finally, you can use nix via `nix-shell -p texliveFull --run "pdflatex resume.tex"`

---

### But what if I want an inefficient but nerdy way to read your resume?

Then you can view it as an cryptographically-signed OCI artifact at [ghcr.io/crosleyzack/resume:latest](ghcr.io/crosleyzack/resume:latest). You can do this using tools like [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane):

```
crane manifest ghcr.io/crosleyzack/resume:latest
```

Or you can use [oci.dag.dev](https://oci.dag.dev/?image=ghcr.io%2Fcrosleyzack%2Fresume%3Alatest) for a graphical interface.

The images are generated from in a github workflow using the code in the `src` directory and are signed with [sigstore cosign](https://github.com/sigstore/cosign). Cosign stores the signature for artifact `ghcr.io/crosleyzack/resume@sha256:[a-fA-F0-9]{64}` as another artifact, `ghcr.io/crosleyzack/resume:sha256-[a-fA-F0-9]{64}.sig`.
