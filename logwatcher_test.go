package logwatcher

import (
	"testing"
	"fmt"
	"io/ioutil"
	"bufio"
	"os"
	"errors"
)

type testFileConfig struct {
	name string
	steps []int
	stepCount int
	size int
	path string
}

var testFiles = []testFileConfig{
	testFileConfig{
		name: "always_empty.log",
		steps: []int{0},
	},
	testFileConfig{
		name: "starts_empty.log",
		steps: []int{ 0, 1, 2, 3 },
	},
	testFileConfig{
		name: "big_static.log",
		steps: []int{65535},
	},
	testFileConfig{
		name: "big_steps.log",
		steps: []int{ 65335, 65335, 65335, 65335 },
	},
	testFileConfig{
		name: "big_steps_start_empty.log",
		steps: []int{ 65335, 65335, 65335, 65335 },
	},
}

var (
	tempDir string
	tempFiles []string
	bogoLine = "127.0.0.1 - - [20/Aug/2015 13:01:03] \"POST /v1/stats HTTP/1.0\" 200 2"
	tfMap map[string]*testFileConfig
)

func setup() {
	var err error
	tempDir, err = ioutil.TempDir("/tmp", "logstest")

	if err != nil {
		panic(fmt.Sprintf("Problem making tempdir: %v", err))
	}

	tfMap = make(map[string]*testFileConfig)
	// Create initial test logfiles
	for i, _ := range testFiles {
		tf := &testFiles[i]
		//tf.makefile(tempDir)
		tfMap[tf.path] = tf
		// if TestDebug {
		// 	common.Log.Debugf("map[%s] = %+v", tf.path, tfMap[tf.path])
		// }
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
//	cleanup()
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

	lw := New(&Config{ Filename: f.Name() })
	scanner := bufio.NewScanner(lw)

	var lines []string
	var c = 0

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
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

	lw := New(&Config{ Filename: f.Name() })
	if err := verifyLines(lw, startLine, nLines, template); err != nil {
		t.Errorf("Error verifying lines %d-%d in %s: %v",
			startLine, startLine + nLines, lw.Filename, err)
	}

	// Truncate
	if err := os.Truncate(lw.Filename, 0); err != nil {
		panic(err)
	}

	// Should read nothing.
	if b, err := nextRead(lw); err != nil {
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

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
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
	}
	if len(lines) != nLines {
		return errors.New(
			fmt.Sprintf("Wrong # of lines %d != %d", len(lines), nLines))
	}
	return nil
}
