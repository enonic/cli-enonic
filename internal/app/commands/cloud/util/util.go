package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/enonic/cli-enonic/internal/app/commands/common"
	"gopkg.in/cheggaaa/pb.v1"
)

// CloudConfigFolder returns the folder of cloud configuration
func CloudConfigFolder() (string, error) {
	folder := filepath.Join(common.GetEnonicDir(), "cloud")
	return folder, os.MkdirAll(folder, os.ModeDir|os.ModePerm)
}

// ReadFile opens a file and pipes it to a reader
func ReadFile(file string, handler func(io.Reader) error) error {
	if err := FileExist(file); err != nil {
		return fmt.Errorf("file '%s' does not exist", file)
	}
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("could not open file '%s': %v", file, err)
	}
	defer f.Close()
	return handler(f)
}

// ReadFileToString reads an entire file into memory
func ReadFileToString(file string) (string, error) {
	var res string
	err := ReadFile(file, func(r io.Reader) error {
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		res = string(b)
		return nil
	})
	return res, err
}

// RemoveFile removes a file if it exists
func RemoveFile(file string) error {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil
	}
	return os.Remove(file)
}

// WriteFile opens a file and uses a writer to write the contents of that file
func WriteFile(file string, perm os.FileMode, write func(io.Writer) error) error {
	// Create a reader/writer pipe
	r, w := io.Pipe()

	// Start writing to the pipe
	var writeErr error
	go func() {
		defer w.Close()
		writeErr = write(w)
	}()

	// Start reading from the pipe to the file
	if err := writeFileWithReader(file, perm, r); err != nil {
		return err
	}

	return writeErr
}

// Util functions

// FileExist checks if file exists
func FileExist(file string) error {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return errors.New("file '" + file + "' does not exist")
	}
	if info.IsDir() {
		return errors.New(file + " is a directory not a file")
	}
	return nil
}

func writeFileWithReader(file string, perm os.FileMode, r io.Reader) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// write a chunk
		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// CreateProgressBar creates a progress bar
func CreateProgressBar(total int64, prefix string) *pb.ProgressBar {
	bar := pb.New(int(total))
	bar.ShowSpeed = false
	bar.ShowCounters = false
	bar.ShowPercent = true
	bar.ShowTimeLeft = false
	bar.ShowElapsedTime = false
	bar.ShowFinalTime = true
	bar = bar.Prefix(prefix).SetUnits(pb.U_BYTES_DEC).SetRefreshRate(200 * time.Millisecond)
	return bar
}

// CreateSpinner creates a spinner
func CreateSpinner(message string) *spinner.Spinner {
	message = message + " "
	spin := spinner.New(spinner.CharSets[26], 300*time.Millisecond)
	spin.Writer = os.Stderr
	spin.Prefix = message
	spin.FinalMSG = "\r" + message + "... " //r fixes empty spaces before final msg on windows
	return spin
}
