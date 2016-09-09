
# golang-logwatcher

`golang-logwatcher` a Go package to watch a logfile that may get truncated or replaced over time.

- It does no parsing, but can easily be wrapped in a `bufio.Scanner` to do so.

## Installation

Install using `go get github.com/foozlevazquez/golang-logwatcher`.

## Usage

### Main Concepts

#### LogWatcher

A `logwatcher.LogWatcher` is used as the main entry-point.

```
import logwatcher

var lw *logwatcher.LogWatcher

lw, err := logwatcher.New(
    &logwatcher.Config{
        Filename: "/var/log/mail.log",
        Log:
