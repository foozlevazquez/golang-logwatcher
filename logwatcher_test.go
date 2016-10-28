package logwatcher

import (
	"testing"
	"fmt"
	"io/ioutil"
	"bufio"
	"os"
	"errors"
	"log"
	"flag"
	"io"
)


var (
	tempDir string
	tempFiles []string
	bogoLine = "127.0.0.1 - - [20/Aug/2015 13:01:03] \"POST /v1/stats HTTP/1.0\" 200 2"

	debugLog *log.Logger
	debug = flag.Bool("debug", false, "print debugging")
)

func setup() {
	var err error

	if !flag.Parsed() {
		flag.Parse()
	}
	if *debug {
		debugLog = log.New(os.Stderr, "", log.LstdFlags)
	}

	tempDir, err = ioutil.TempDir("/tmp", "logstest")

	if err != nil {
		panic(fmt.Sprintf("Problem making tempdir: %v", err))
	}

}

func cleanup() {
	if err := os.RemoveAll(tempDir); err != nil {
		fmt.Printf("Error deleting %s: %v", tempDir, err)
	}
}

func TestMain(m *testing.M) {
	setup()
	rc := m.Run()
	cleanup()
	os.Exit(rc)
}

// openTempFile opens a random file in dir directory for writing, and
// returns the *os.File.
//
// openTempFile panics on errors.
func openTempFile(dir string) (f *os.File) {
	if f, err := ioutil.TempFile(dir, "random-"); err != nil {
		panic(err)
	} else {
		return f
	}
}

func mustOpen(filename string) *os.File {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	return f
}

func TestSimplest(t *testing.T) {

	f := openTempFile(tempDir)
	nLines := 10
	template := "%d log entry!"

	for i := 0; i < nLines; i++ {
		_, err := fmt.Fprintf(f, template + "\n", i)
		if err != nil {
			panic(fmt.Sprintf("Error writing line %d to %s: %v",
				i, f.Name, err))
		}
	}
	if err := f.Close(); err != nil {
		panic(fmt.Sprintf("Error closing %s: %v", f.Name(), err))
	}

	lw := New(&Config{ Filename: f.Name(), Log: debugLog})
	scanner := bufio.NewScanner(lw)

	var lines []string
	var c = 0

	for scanner.Scan() {
		nl := scanner.Text()
		lines = append(lines, nl)
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if lines[c] != fmt.Sprintf(template, c) {
			t.Errorf("Bad line %q at line %d", lines[c], c)
		}
		c += 1
	}
	if len(lines) != nLines {
		t.Errorf("Wrong # of lines %d != %d", len(lines), nLines)
	}
}


func writeLines(f *os.File, startLine int, nLines int, template string) {

	for i := 0; i < nLines; i++ {
		ln := i + startLine
		_, err := fmt.Fprintf(f, template + "\n", ln)
		if err != nil {
			panic(fmt.Sprintf("Error writing line %d to %s: %v",
				ln, f.Name, err))
		}
	}
}

func mustClose(f *os.File) {
	if err := f.Close(); err != nil {
		panic(fmt.Sprintf("Error closing %s: %v", f.Name(), err))
	}
}

func TestTruncation(t *testing.T) {
	template := "%d log entry!"

	startLine := 0
	nLines := 100

	f := openTempFile(tempDir)
	writeLines(f, startLine, nLines, template)
	mustClose(f)

	lw := New(&Config{ Filename: f.Name(), Log: debugLog })
	if err := verifyLines(lw, startLine, nLines, template); err != nil {
		t.Errorf("Error verifying lines %d-%d in %s: %v",
			startLine, startLine + nLines, lw.Filename, err)
	}

	// Truncate
	if err := os.Truncate(lw.Filename, 0); err != nil {
		panic(err)
	}

	// Should read nothing.
	if b, err := nextRead(lw); err != nil && err != io.EOF {
		panic(err)
	} else if len(b) != 0 {
		t.Errorf("Error reading 0 bytes after truncation.")
	}

	startLine = nLines
	nLines = 10
	template = "%d LOG ENTRY!"
	// Write continued lines to truncated file.
	f = mustOpen(lw.Filename)
	writeLines(f, startLine, nLines, template)
	mustClose(f)

	if err := verifyLines(lw, startLine, nLines, template); err != nil {
		t.Errorf("Error verifying lines %d-%d in truncated %s: %v",
			startLine, startLine + nLines, lw.Filename, err)
	}
	// Should read nothing.
	if b, err := nextRead(lw); err != nil && err != io.EOF {
		panic(err)
	} else if len(b) != 0 {
		t.Errorf("Error reading 0 bytes after truncation.")
	}
}

func TestRestartFilepos(t *testing.T) {
	template := "%d restarted log entry!"

	// Write 50 lines
	startLine := 0
	nLines := 50

	f := openTempFile(tempDir)
	writeLines(f, startLine, nLines, template)
	mustClose(f)

	// record position
	pos, err := fileSize(f.Name())
	if err != nil { panic(err) }

	// write next 50 lines
	f = mustOpen(f.Name())
	startLine = 50
	writeLines(f, startLine, nLines, template)

	// Verify, starting at line 50 position.
	lw := New(
		&Config{
			Filename: f.Name(),
			Log: debugLog,
			StartPosition: pos,
		})
	if err := verifyLines(lw, startLine, nLines, template); err != nil {
		t.Errorf("Error verifying lines %d-%d in %s: %v",
			startLine, startLine + nLines, lw.Filename, err)
	}

	// Should read nothing.
	if b, err := nextRead(lw); err != nil && err != io.EOF {
		panic(err)
	} else if len(b) != 0 {
		t.Errorf("Error reading 0 bytes after truncation.")
	}

}


func nextRead(lw *LogWatcher) ([]byte, error) {
	var b = []byte{}

	if _, err := lw.Read(b); err != nil {
		return nil, err
	} else {
		return b, nil
	}
}

func verifyLines(lw *LogWatcher, startLine, nLines int, template string) error {
	var lines []string
	var c = 0
	var scanner = bufio.NewScanner(lw)

	for done := false; !done; {
		// Scan returns false when Scan is done.
		if scanner.Scan() {
			ln := scanner.Text()
			//fmt.Printf("Scanner read %q\n", ln)
			lines = append(lines, ln)
			if err := scanner.Err(); err != nil {
				return errors.New(
					fmt.Sprintf("Error scanning line %d of %s: %v",
						c, lw.Filename, err))
			}
			needLine := fmt.Sprintf(template, c + startLine)
			if lines[c] != needLine {
				return errors.New(fmt.Sprintf("Bad line %q != %q at line %d",
					lines[c], needLine, c))
			}
			c += 1
		} else {
			done = true
		}
	}
	if len(lines) != nLines {
		return errors.New(
			fmt.Sprintf("Wrong # of lines %d != %d", len(lines), nLines))
	}
	return nil
}


func TestResetLastState(t *testing.T) {

	f := openTempFile(tempDir)
	nLines := 10
	template := "%d log entry!"
	knownSize := 0

	for i := 0; i < nLines; i++ {
		s := fmt.Sprintf(template + "\n", i)
		knownSize += len(s)
		_, err := f.Write([]byte(s))
		if err != nil {
			panic(fmt.Sprintf("Error writing line %d to %s: %v",
				i, f.Name, err))
		}
	}
	if err := f.Close(); err != nil {
		panic(fmt.Sprintf("Error closing %s: %v", f.Name(), err))
	}

	lw := New(&Config{ Filename: f.Name(), Log: debugLog})

	if err := lw.ResetLastState(); err != nil {
		t.Error(err)
	}
	size, err := lw.Size()
	if err != nil {
		t.Error(err)
	}

	if size != int64(knownSize) {
		t.Errorf("Mismatching sizes %d (expected) != %d (got)",
			knownSize, size)
	}

	if err = lw.SetLastPosition(size); err != nil {
		t.Error(err)
	}
	if lw.LastPosition() != size {
		t.Errorf("Incorrect last position.")
	}
}

func fileSize(filename string) (int64, error) {
	if fInfo, err := os.Stat(filename); err != nil {
		return 0, err
	} else {
		return fInfo.Size(), nil
	}
}
