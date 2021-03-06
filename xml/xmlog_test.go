package xmlog

import (
	"os"
	"time"
	"runtime"
	"io/ioutil"
	"fmt"

	l4g "github.com/ccpaging/log4go"
	"testing"
)

const testLogFile = "_logtest.log"

var now time.Time = time.Unix(0, 1234567890123456789).In(time.UTC)

func newLogRecord(lvl l4g.Level, src string, msg string) *l4g.LogRecord {
	return &l4g.LogRecord{
		Level:   lvl,
		Source:  src,
		Created: now,
		Message: msg,
	}
}

func TestXMLogWriter(t *testing.T) {
	defer func(buflen int) {
		l4g.DefaultBufferLength = buflen
	}(l4g.DefaultBufferLength)
	l4g.DefaultBufferLength = 0

	w := NewLogWriter(testLogFile, 0)
	if w == nil {
		t.Fatalf("Invalid return: w should not be nil")
	}
	defer os.Remove(testLogFile)

	w.LogWrite(newLogRecord(l4g.CRITICAL, "source", "message"))
	w.Close()
	runtime.Gosched()

	if contents, err := ioutil.ReadFile(testLogFile); err != nil {
		t.Errorf("read(%q): %s", testLogFile, err)
	} else if len(contents) != 170 {
		t.Errorf("malformed xml log: %q (%d bytes)", string(contents), len(contents))
	}
}

func TestXMLConfig(t *testing.T) {
	const (
		configfile = "_example.xml"
	)

	fd, err := os.Create(configfile)
	if err != nil {
		t.Fatalf("Could not open %s for writing: %s", configfile, err)
	}

	fmt.Fprintln(fd, 
`<logging>
	<filter enabled="true">
	    <tag>stdout</tag>
	    <type>console</type>
	    <!-- level is (:?FINEST|FINE|DEBUG|TRACE|INFO|WARNING|ERROR) -->
	    <level>DEBUG</level>
		<property name="format">[%D %T] [%L] (%S) %M</property>
	</filter>
	<filter enabled="true">
		<tag>file</tag>
		<type>file</type>
		<level>FINEST</level>
		<property name="filename">_test.log</property>
		<!--
			%T - Time (15:04:05 MST)
			%t - Time (15:04)
			%D - Date (2006/01/02)
			%d - Date (01/02/06)
			%L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
			%S - Source
			%s - Short Source
			%x - Extra Short Source: just file without .go suffix
			%M - Message
			It ignores unknown format strings (and removes them)
			Recommended: "[%D %T] [%L] (%S) %M"
		-->
		<property name="format">[%D %T] [%L] (%S) %M</property>
		<property name="flush">0</property> <!-- no bufio write -->
		<property name="maxrotate">0</property> <!-- enables log rotation. 0, append -->
		<property name="maxsize">0M</property> <!-- \d+[KMG]? Suffixes are in terms of 2**10 -->
		<property name="cycle">24h</property> <!-- rotate cycle -->
		<property name="delay0">0h</property> <!-- rotate clock since midnight -->
	</filter>
	<filter enabled="true">
		<tag>xml</tag>
		<type>xml</type>
		<level>TRACE</level>
		<property name="filename">_trace.xml</property>
		<property name="flush">4k</property> <!-- bufio flush size -->
		<property name="maxrotate">10</property> <!-- enables log rotation. 0, append -->
		<property name="maxsize">10M</property> <!-- \d+[KMG]? Suffixes are in terms of 2**10 -->
		<property name="cycle">24h</property> <!-- rotate cycle -->
		<property name="delay0">0h</property> <!-- rotate clock since midnight -->
	</filter>
	<filter enabled="false"><!-- enabled=false means this logger won't actually be created -->
		<tag>donotopen</tag>
		<type>socket</type>
		<level>FINEST</level>
		<property name="endpoint">192.168.1.255:12124</property> <!-- recommend UDP broadcast -->
		<property name="protocol">udp</property> <!-- tcp or udp -->
		<property name="format">[%D %T] [%L] (%S) %M</property>
	</filter>
</logging>`)
	fd.Close()

	log := make(l4g.Logger)
	LoadConfiguration(log, configfile)
	defer os.Remove("_trace.xml")
	defer os.Remove("_test.log")
	defer log.Close()

	// Make sure we got all loggers
	if len(log) != 3 {
		t.Fatalf("XMLConfig: Expected 3 filters, found %d", len(log))
	}

	// Make sure they're the right keys
	if _, ok := log["stdout"]; !ok {
		t.Errorf("XMLConfig: Expected stdout logger")
	}
	if _, ok := log["file"]; !ok {
		t.Fatalf("XMLConfig: Expected file logger")
	}
	if _, ok := log["xml"]; !ok {
		t.Fatalf("XMLConfig: Expected xml logger")
	}

	// Make sure they're the right type
	if _, ok := log["stdout"].LogWriter.(*l4g.ConsoleLogWriter); !ok {
		t.Fatalf("XMLConfig: Expected stdout to be ConsoleLogWriter, found %T", log["stdout"].LogWriter)
	}
	if _, ok := log["file"].LogWriter.(*l4g.FileLogWriter); !ok {
		t.Fatalf("XMLConfig: Expected file to be *FileLogWriter, found %T", log["file"].LogWriter)
	}
	if _, ok := log["xml"].LogWriter.(*l4g.FileLogWriter); !ok {
		t.Fatalf("XMLConfig: Expected xml log to be *FileLogWriter, found %T", log["xml"].LogWriter)
	}

	// Make sure levels are set
	if lvl := log["stdout"].Level; lvl != l4g.DEBUG {
		t.Errorf("XMLConfig: Expected stdout to be set to level %d, found %d", l4g.DEBUG, lvl)
	}
	if lvl := log["file"].Level; lvl != l4g.FINEST {
		t.Errorf("XMLConfig: Expected file to be set to level %d, found %d", l4g.FINEST, lvl)
	}
	if lvl := log["xml"].Level; lvl != l4g.TRACE {
		t.Errorf("XMLConfig: Expected xml log to be set to level %d, found %d", l4g.TRACE, lvl)
	}

	// Make sure the w is open and points to the right file
	/*
	flw := log["file"].LogWriter.(*l4g.FileLogWriter)
	if fname, _ := flw.GetOption("filename"); fname != "_test.log" {
		t.Errorf("XMLConfig: Expected file to have opened %s, found %s", "test.log", fname)
	}

	xlw := log["xml"].LogWriter.(*l4g.FileLogWriter)
	if fname, _ := xlw.GetOption("filename"); fname != "_trace.xml" {
		t.Errorf("XMLConfig: Expected file to have opened %s, found %s", "_trace.xml", fname)
	}
	*/

	// Save XML log file
	err = os.Rename(configfile, "example/config.xml") // Keep this so that an example with the documentation is available
	if err != nil {
		t.Fatalf("Could not rename %s: %s", configfile, err)
	}
	os.Remove(configfile)
}
