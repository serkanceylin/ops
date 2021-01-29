package mkfs

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-errors/errors"
)

// MkfsCommand wraps mkfs calls
type MkfsCommand struct {
	bootPath   string
	label      string
	manifest   *Manifest
	size       int64
	outPath	   string
}

// NewMkfsCommand returns an instance of MkfsCommand
func NewMkfsCommand(m *Manifest) *MkfsCommand {
	return &MkfsCommand{
		bootPath:   "",
		label:      "",
		manifest:   m,
		targetRoot: "",
		size:       0,
		outPath:    "",
	}
}

// SetEmptyFileSystem add argument that sets file system as empty
func (m *MkfsCommand) SetEmptyFileSystem() {
	m.manifest = nil
}

// SetFileSystemSize adds argument that sets file system size
func (m *MkfsCommand) SetFileSystemSize(size string) error {
	f := func(c rune) bool {
		return !unicode.IsNumber(c)
	}
	unitsIndex := strings.IndexFunc(size, f)
	var mul int64
	if unitsIndex < 0 {
		mul = 1
		unitsIndex = len(size)
	} else if unitsIndex == 0 {
		return errors.New("invalid size " + size)
	} else {
		units := strings.ToLower(size[unitsIndex:])
		if units == "k" {
			mul = 1024
		} else if units == "m" {
			mul = 1024 * 1024
		} else if units == "g" {
			mul = 1024 * 1024 * 1024
		} else {
			return errors.New("invalid units " + units)
		}
	}
	var err error
	m.size, err = strconv.ParseInt(size[:unitsIndex], 10, 64)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	m.size *= mul
	return nil
}

// SetBoot adds argument that sets file system boot
func (m *MkfsCommand) SetBoot(boot string) {
	m.bootPath = boot
}

// SetFileSystemPath add argument that sets file system path
func (m *MkfsCommand) SetFileSystemPath(fsPath string) {
	m.outPath = fsPath
}

// SetLabel add label argument that sets file system label
func (m *MkfsCommand) SetLabel(label string) {
	m.label = label
}

// Execute runs mkfs command
func (m *MkfsCommand) Execute() error {
	if m.outPath == "" {
		return fmt.Errorf("output image file path not set")
	}
	var outFile *os.File
	var err error
	outFile, err = os.Create(m.outPath)
	if err != nil {
		return fmt.Errorf("cannot create output file %s: %v", m.outPath, err)
	}
	defer outFile.Close()
	var rootFSOffset int64 = 0
	var bootFile *os.File
	if m.bootPath != "" {
		bootFile, err = os.Open(m.bootPath)
		if err != nil {
			return fmt.Errorf("cannot open boot image %s: %v", m.bootPath, err)
		}
		defer bootFile.Close()
		b := make([]byte, 8192)
		for {
			n, err := bootFile.Read(b)
		    if err == io.EOF {
		    	break;
		    } else if err != nil {
		    	return fmt.Errorf("cannot read boot image %s: %v", m.bootPath, err)
		    }
			n, err = outFile.Write(b[:n])
		    if err != nil {
		    	return fmt.Errorf("cannot write output file %s: %v", m.outPath, err)
		    }
		    rootFSOffset += n
		}
	}
	if m.manifest == nil {
		m.manifest = NewManifest("")
	}
	if len(m.manifest.boot) != 0 {
		rootFSOffset += KLOG_DUMP_SIZE;
	}
	return nil
}

// GetUUID returns the uuid of file system built
func (m *MkfsCommand) GetUUID() string {
	return ""
}
