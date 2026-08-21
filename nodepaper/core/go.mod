module nodepaper

go 1.24

// Pin the toolchain, not just the language version. Release candidates were
// being built on whichever Go happened to be installed: rc.2 and rc.8 came out
// of CI on 1.26.5, rc.3 through rc.7 off a workstation on 1.26.4, and
// release-manifest recorded whichever it got without anyone deciding. The CI
// workflows already say 1.26.5, so this makes a local build agree with them
// instead of quietly differing.
toolchain go1.26.5

require gopkg.in/yaml.v3 v3.0.1
