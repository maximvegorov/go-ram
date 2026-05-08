package ram

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

type Input interface {
	Read() (int, error)
}

func NewInput(r io.Reader) Input {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	return readerInput{scanner: scanner}
}

type Output interface {
	Write(v int) error
	Flush() error
}

func NewOutput(w io.Writer) Output {
	return writerOutput{writer: bufio.NewWriter(w)}
}

type readerInput struct {
	scanner *bufio.Scanner
}

func (r readerInput) Read() (int, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return 0, err
		}
		return 0, io.EOF
	}
	return strconv.Atoi(r.scanner.Text())
}

type writerOutput struct {
	writer *bufio.Writer
}

func (w writerOutput) Flush() error {
	return w.writer.Flush()
}

func (w writerOutput) Write(v int) error {
	_, err := fmt.Fprintln(w.writer, v)
	return err
}
