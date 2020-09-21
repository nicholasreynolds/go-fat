package disk

import (
	"os"
	"testing"
)

func TestFile_Read(t *testing.T) {

}

func TestFile_ReadAt(t *testing.T) {

}

func TestFile_Write(t *testing.T) {

}

func TestFile_WriteAt(t *testing.T) {

}

func TestFile_Close(t *testing.T) {
	// Setup
	tDiskFilename, tBlockCt := "test.disk", 64
	tFilename := "test.txt"
	d, err := New(tDiskFilename, tBlockCt)
	if err != nil {
		t.Error(err)
	}
	var f File
	f, err = d.Create(tFilename)
	if err != nil {
		t.Error(err)
	}
	// Test
	f.Close()
	if f.disk.checkIsOpen(tFilename) {
		t.Errorf("Filename not closed: %s", tFilename)
	}
	// Teardown
	f.disk.fd.Close()
	os.Remove(tDiskFilename)
}