package logwatcher

import (
	"os"
	"errors"
	"log"
	"io"
	"time"
	"syscall"
	"fmt"
)

type Config struct {
	// Filename of the logfile to watch.
	Filename string

	// Where to start reading, the first time we open this, if the DeviceID
	// and FileID match as well.
	StartPosition	int64

	// What device we're on (Unix specific I know).
	DeviceID		int32

	// What file we last were reading (Inode).
	FileID			uint64

	// Where to write messages to, if left nil, then debugging messages are
	// discarded.
	Log *log.Logger

}

func (lw *LogWatcher) debugf(s string, v ... interface{}) {
	if lw.Log != nil {
		lw.Log.Printf("[DEBUG] " + s, v...)
	}
}

type LogWatcher struct {
	Config
	lastPos   int64
	lastFInfo os.FileInfo
}

func (lw *LogWatcher) SetLastPosition(p int64) error {
	size, err := lw.Size()
	if err != nil {
		return err
	}
	if p > size {
		return errors.New(
			fmt.Sprintf("logwatcher: SetLastPosition(%d) past size (%d)",
				p, size))
	}
	lw.lastPos = p
	return nil
}

// ResetLastState makes the LogWatcher last status current to whatever state
// the file is in. It does not update the LastPosition.

func (lw *LogWatcher) ResetLastState() error {
	if fInfo, err := os.Stat(lw.Filename); err != nil {
		return err
	} else {
		lw.lastFInfo = fInfo
		return nil
	}
}

func (lw *LogWatcher) LastPosition() int64 {
	return lw.lastPos
}

func (lw *LogWatcher) Size() (int64, error) {
	if lw.lastFInfo != nil {
		return lw.lastFInfo.Size(), nil
	} else {
		return 0, errors.New("No current state of file")
	}
}

func (lw *LogWatcher) ModTime() (time.Time, error) {
	if lw.lastFInfo != nil {
		return lw.lastFInfo.ModTime(), nil
	} else {
		return time.Time{}, errors.New("No current state of file")
	}
}

func (lw *LogWatcher) DeviceID() (uint64, error) {
	if lw.lastFInfo != nil {
		return lw.lastFInfo.Sys().(syscall.Stat_t).Dev, nil
	} else {
		return 0, errors.New("No current state of file")
	}
}

func (lw *LogWatcher) Inode() (uint64, error) {
	if lw.lastFInfo != nil {
		return lw.lastFInfo.Sys().(syscall.Stat_t).Ino, nil
	} else {
		return 0, errors.New("No current state of file")
	}
}

func New(config *Config) *LogWatcher {
	lw := &LogWatcher{Config: *config}
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
// Read will only actually open and read the underlying logfile if there are
// indications that there is new data to be read.
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

		lw.debugf("logwatcher.Read: fInfo: %+v", fInfo)

		if lw.lastFInfo == nil {
			newFile = true
			// User can pass a checkpointed position.
			lw.lastPos = lw.StartPosition
			lw.debugf("logwatcher.Read: newfile, lastpos = %d", lw.lastPos)
		} else if !os.SameFile(lw.lastFInfo, fInfo) {
			newFile = true
			lw.debugf("logwatcher.Read: not samefile.")
		} else if fInfo.Size() < lw.lastFInfo.Size() {
			// Truncated
			lw.lastPos = 0
			newFile = true
			lw.debugf("logwatcher.Read: truncated.")
		} else if fInfo.Size() > lw.lastFInfo.Size() {
			// logfile grew, read it
			doRead = true
			lw.debugf("logwatcher.Read: bigger file reading.")
		} else if fInfo.Size() > lw.lastPos {
			// logfile grew, read it
			doRead = true
			lw.debugf("logwatcher.Read: stuff left to read, reading.")
		} else {
			// same size, don't read
			lw.debugf("logwatcher.Read: no change, ignoring.")
			err = io.EOF
		}

		if newFile && fInfo.Size() > 0 {
			// Reset pointers
			lw.lastFInfo = nil
			doRead = true
			lw.debugf("logwatcher.Read: bigger file reading.")
		}

		if doRead {
			return lw.read(fInfo, buf)
		}
	}
	lw.debugf("logwatcher.Read: Returning 0, %v", err)
	return 0, err
}

// read does the underlying work of reading the log file and updating data to
// keep track of where we've read up to.
func (lw *LogWatcher) read(fInfo os.FileInfo, buf []byte) (int, error) {
	f, err := os.Open(lw.Filename)
	if err != nil {
		return 0, err
	}

	lw.debugf("logwatcher.read: %+v", lw)
	if lw.lastPos > 0 {
		// seek to last position read
		lw.debugf("logwatcher.read: %q seeking to %d", lw.Filename,
			lw.lastPos)

		if _, err := f.Seek(lw.lastPos, 0); err != nil {
			return 0, ErrSeek
		}
	}

	n, err := f.Read(buf)
	if err != nil {
		lw.debugf("logwatcher.read: err = %v", err)
		return 0, err
	}

	lw.debugf("logwatcher.read: %q read %d: %q", lw.Filename,
		n, string(buf[0:n]))
	lw.lastFInfo = fInfo
	lw.lastPos += int64(n)

	if err = f.Close(); err != nil {
		return n, err
	}
	return n, nil
}
