package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

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

func RemoveFile(file string) error {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil
	}
	return os.Remove(file)
}

func WriteFile(file string, perm os.FileMode, write func(io.Writer) error) error {
	r, w := io.Pipe()

	var writeErr error
	go func() {
		writeErr = writeFileWithReader(file, perm, r)
		r.Close()
	}()

	if err := write(w); err != nil {
		return err
	}

	w.Close()

	return writeErr
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
			panic(err)
		}
		if n == 0 {
			break
		}

		// write a chunk
		if _, err := f.Write(buf[:n]); err != nil {
			panic(err)
		}
	}
	return nil
}
