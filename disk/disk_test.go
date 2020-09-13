package disk

import (
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	// Setup
	filename, block_ct := "test.disk", 64
	// Preliminary non-exported function tests
	t.Run("newDisk", func(t *testing.T) {
		d, err := newDisk(filename, block_ct)
		if err != nil {
			// Covers any file-related kernel and i/o errors
			t.Error(err)
		}
		if d.dataBlks != block_ct {
			t.Errorf("Expected %v data blocks, Got %v", block_ct, d.dataBlks)
		}
		//Teardown
		os.Remove(filename)
	})
	t.Run("initFS", func(t *testing.T) {
		// Setup
		d, _ := newDisk(filename, block_ct)
		// Test
		d.initFS()
		file := d.fd
		if file == nil {
			t.Error("Nil disk file %s found")
		}
		fLenExp := 2*BlockSize + 2*block_ct + block_ct*BlockSize
		fLenGot, _ := file.Stat()
		if fLenGot.Size() < int64(fLenExp) {
			t.Error("Expected disk size %v, Got %v", )
		}
		// Teardown
		os.Remove(filename)
	})
	t.Run("initSuperblock", func(t *testing.T) {
		// Setup
		d, _ := newDisk(filename, block_ct)
		// Test
		d.initSuperblock()
		// Teardown
		os.Remove(filename)
	})
	// Test
	_, err := Create(filename, block_ct)
	if err != nil {
		t.Error(err)
	}
	// Teardown
	os.Remove(filename)
}
