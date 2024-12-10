---
title: "Living off the land in Linux!"
description: "I developed a Go program to exploit misconfigurations in Linux binaries based on https://gtfobins.github.io"
date: "2024-12-03"
---

Last year a coworker shared with me the excellent https://gtfobins.github.io/ website, a compendium of techniques to abuse misconfigurations in Linux binaries.
It's an open source project and its code can be found on GitHub at https://github.com/GTFOBins/GTFOBins.github.io.

I wanted to build a discovery tool leveraging this data set.
I had some time on a week-end and created https://github.com/juliendoutre/gogtfobins/ to do so.
It's a Go CLI built with the [cobra library](https://github.com/spf13/cobra) exposing three commands:
- `gogtfobins list` to list all binaries available on the host and the functions they can eventually allow to obtain
- `gogtfobins describe BINARY` to print some details about a specific binary
- `gogtfobins exploit BINARY FUNCTION` to run an exploit for a binary

Here is a more concrete example:
```shell
# List all available binaries allowing for opening a reverse shell on the current host.
gogtfobins list --function reverse-shell
# Print possible exploits for the docker binary.
gogtfobins describe docker
# Get a reverse-shell using the docker binary.
gogtfobins exploit docker reverse-shell
```

You can install it easily with [homebrew](https://brew.sh/):
```shell
brew tap juliendoutre/gogtfobins https://github.com/juliendoutre/gogtfobins
brew install gogtfobins
```

or download a built binary from the available [releases](https://github.com/juliendoutre/gogtfobins/releases).

The gtfobins data is embedded thanks to a [go embed](https://pkg.go.dev/embed) directive that is then used to build both [an index and a reverse index](https://github.com/juliendoutre/gogtfobins/blob/main/index.go).
Commands simply query this index. You can actually reuse this data structure in your own project as it is exposed in a Go module:
```shell
go get github.com/juliendoutre/gogtfobins
```

Let me know if anything is missing or you found a bug by opening [an issue](https://github.com/juliendoutre/gogtfobins/issues).

See you next time :wave:
