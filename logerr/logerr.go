package logerr

//Tämä on peräisin GO standard librarysta (log.go). Mutex/sync sekä aikaleimat
// poistettu  sekä muutenkin raskaasti modifioitu bikeride tarpeisiin.

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const warningPrefix = "***warning: "
const errorPrefix = "***error: "

// A logger represents an active logging object that generates lines of
// output to an io.Writer. Each logging operation makes a single call to
// the Writer's Write method.
type logger struct {
	prefix string // prefix to write at beginning of each line
	out io.Writer // destination for output
	buf []byte    // for accumulating text to write
}

// Logerr wraps two loggers and adds error count,
// event level and output mode option (Stdout and/or file).
type Logerr struct {
	console *logger
	file    *logger
	level   int
	mode    int
	errors  int
}

// New creates a new Logerr with two loggers initialized to Stdout and Stderr
func New() *Logerr {
	return &Logerr{
		level:   1,
		mode:    0,
		console: &logger{out: os.Stdout},
		file:    &logger{out: os.Stderr}}
}

func (e *Logerr) TimeStamp() string {
	return fmt.Sprintf("%v", time.Now())[0:23]
}

func (e *Logerr) TimeNow() time.Time {
	return time.Now()
}

func (e *Logerr) TimeSince(start time.Time) time.Duration {
	return time.Since(start)
}

func (e *Logerr) SetPrefix(prefix string) {
	e.file.prefix = prefix
	e.console.prefix = prefix
}

func (e *Logerr) Prefix() string {
	return e.file.prefix
}

func (e *Logerr) SetLevel(i int) {
	e.level = i
}

func (e *Logerr) Level() int {
	return e.level
}

func (e *Logerr) SetMode(i int) {
	e.mode = i
}

func (e *Logerr) Mode() int {
	return e.mode
}

func (e *Logerr) SetOutput(filename string) error {

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	e.file.out = f
	return nil
}

func (e *Logerr) Err(v ...interface{}) error {

	defer e.SetPrefix(e.file.prefix)
	e.SetPrefix(errorPrefix)

	e.errors++
	msg := fmt.Sprintln(v...)
	if err := e.console.output(msg); err != nil {
		return err
	}
	if e.mode != 1 {
		return nil
	}
	return e.file.output(msg)
}

func (e *Logerr) Msg(level int, v ...interface{}) error {

	if e.mode < 0 || level > e.level {
		return nil
	}
	msg := fmt.Sprintln(v...)
	return e.Output(level, msg)
}

func (e *Logerr) SegMsg(level, segment int, v ...interface{}) error {

	if e.mode < 0 || level > e.level {
		return nil
	}
	seg := strconv.FormatInt(int64(segment), 10)
	switch len(seg) {
	case 1:
		seg = "   " + seg
	case 2:
		seg = "  " + seg
	case 3:
		seg = " " + seg
	}
	msg := "seg " + seg +  "\t" + fmt.Sprintln(v...)
	return e.Output(level, msg)
}

func (e *Logerr) Errorf(format string, v ...interface{}) error {
	return fmt.Errorf(format, v...)
}

func (e *Logerr) Printf(format string, v ...interface{}) (int, error) {
	return fmt.Fprintf(e.console.out, format, v...)
}

func (e *Logerr) Println(v ...interface{}) (int, error) {
	return fmt.Println(v...)
}

func (e *Logerr) Sprintf(format string, v ...interface{}) string {
	return fmt.Sprintf(format, v...)
}

func (e *Logerr) Errors() int {
	return e.errors
}

func (e *Logerr) Output(level int, s string) error {

	if level == 0 {
		defer e.SetPrefix(e.file.prefix)
		e.SetPrefix(warningPrefix)
	}
	if e.mode >= 0 && level == 0 {
		if err := e.console.output(s); err != nil {
			return err
		}
	}
	if e.mode == 1 && level <= e.level {
		return e.file.output(s)
	}
	if e.mode == 2 && 0 < level && level <= e.level {
		return e.console.output(s)
	}
	return nil
}

func (l *logger) output(s string) error {

	l.buf = l.buf[:0]
	l.buf = append(l.buf, l.prefix...)
	l.buf = append(l.buf, s...)

	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}
