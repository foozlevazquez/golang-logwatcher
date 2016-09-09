

# golang-logwatcher

`golang-logwatcher` a package fo Go to watch a logfile that changes over time
and sends updates to a listener.

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
