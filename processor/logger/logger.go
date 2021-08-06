package logger

import (
	"bufio"
	"bytes"
	"go.uber.org/zap"
	"io"
)

type Logger struct {
	pipelineId string
	*zap.Logger
	remainingString string
}

func NewLogger(pipelineId string, logger *zap.Logger) *Logger {
	return &Logger{pipelineId: pipelineId, Logger: logger}
}

// 参考 bufio.ScanLines，若无法以 \n 分割，则以 \r 分割
//
// ScanLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one optional carriage return followed
// by one mandatory newline. In regular expression notation, it is `\r?\n`.
// The last non-empty line of input will be returned even if it has no
// newline.
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

// 参考 bufio.dropCR
// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func (l *Logger) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	var lineEnd bool

	if p[len(p)-1] == '\n' || p[len(p)-1] == '\r' {
		lineEnd = true
	}

	inputReader := bytes.NewReader(p)
	r := bufio.NewScanner(inputReader)

	r.Split(ScanLines)

	var lastLine string

	for r.Scan() {
		if lastLine != "" {
			l.remainingString = ""
			l.Info(l.remainingString + lastLine)
		}
		lastLine = r.Text()
	}

	if !lineEnd {
		l.remainingString += lastLine
	} else {
		if lastLine != "" {
			l.Info(lastLine)
		}
	}

	return len(p), nil
}

func (l *Logger) ReadFrom(reader io.Reader) (n int64, err error) {
	bufReader := bufio.NewReader(reader)

	for {
		out, _, err := bufReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return n, err
		} else {
			n += int64(len(out))
			l.Logger.Info(string(out))
		}
	}
}
