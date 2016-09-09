package logwatcher

import (
	"os"
	"time"
	"errors"
	"io/ioutil"
)

type Config struct {
	// Filename of the logfile to watch.
	Filename string

	// Where to write messages to.
	Log SimpleLogger

	// read() will block if there is nothing to be read, this is how long to
	// wait before re-checking internally to see if there is new data.
	PollTime time.Duration
}

type LogWatcher struct {
	Config
	lastPos   int64
	lastFInfo os.FileInfo
}

func New(config *Config) *LogWatcher {
	lw := &LogWatcher{Config: *config}
	if lw.Log == nil {
		// Use stderr if none supplied
		lw.Log = &WriterLogger{s: ioutil.Discard}
	}
	return lw
}

// Errors returned by LogWatcher.
var (
	ErrSeek = errors.New("logwatcher: seek error.")
)

// Read tries to fill buf with data from the log file.  Read returns the
// number of bytes read and an error.
//
// Read does not look for newlines, but since LogWatcher conforms to the
// io.Reader interface, a LogWatcher can be wrapped in a bufio.Scanner to
// parse the lines.
//
// The gating factor of how much to read is controlled by the size of buf.
//
// NB: Read does not try to finish reading a logfile after it has been moved,
// it moves to the new logfile.  Therefore a user of LogWatcher should keep
// times between calls to Read short to avoid missing data.
//
func (lw *LogWatcher) Read(buf []byte) (int, error) {
	var err error
	var fInfo os.FileInfo

	if fInfo, err = os.Stat(lw.Filename); err == nil {
		doRead := false
		newFile := false

		if lw.lastFInfo == nil {
			newFile = true
		} else if !os.SameFile(lw.lastFInfo, fInfo) {
			newFile = true
		} else if fInfo.Size() < lw.lastFInfo.Size() {
			// Truncated
			newFile = true
		} else if fInfo.Size() > lw.lastFInfo.Size() {
			// logfile grew, read it
			doRead = true
		} // else same size, don't read

		if newFile && fInfo.Size() > 0 {
			// Reset pointers
			lw.lastPos = 0
			lw.lastFInfo = nil
			doRead = true
		}

		if doRead {
			return lw.read(fInfo, buf)
		}
	}
	return 0, err
}

// read does the underlying work of reading the log file and updating data to
// keep track of where we've read up to.
func (lw *LogWatcher) read(fInfo os.FileInfo, buf []byte) (int, error) {
	f, err := os.Open(lw.Filename)
	if err != nil {
		return 0, err
	}
	lw.Log.Debugf("logwatcher.read: %+v", lw)
	if lw.lastFInfo != nil && lw.lastPos > 0 {
		// seek to last position read
		lw.Log.Debugf("logwatcher.read: %q seeking to %d", lw.Filename,
			lw.lastPos)

		if _, err := f.Seek(lw.lastPos, 0); err != nil {
			return 0, ErrSeek
		}
	}

	n, err := f.Read(buf)
	if err != nil {
		return 0, err
	}

	lw.Log.Debugf("logwatcher.read: %q read %d: %q", lw.Filename,
		n, string(buf[0:n]))
	lw.lastFInfo = fInfo
	lw.lastPos += int64(n)

	if err = f.Close(); err != nil {
		return n, err
	}
	return n, nil
}
