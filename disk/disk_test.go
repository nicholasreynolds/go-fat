package disk

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	// Setup
	tFilename, tBlockCt := "test.disk", 64
	// Internal tests
	t.Run("createDisk", func(t *testing.T) {
		d, err := createDisk(tFilename, tBlockCt)
		if err != nil {
			// Covers any file-related kernel and i/o errors
			t.Error(err)
		}
		if d.fd == nil {
			t.Errorf("Nil file descriptor for '%s'", tFilename)
		}
		if d.dataBlockCt != tBlockCt {
			t.Errorf("Expected %v data blocks, Got %v", tBlockCt, d.dataBlockCt)
		}
		//Teardown
		d.fd.Close()
		os.Remove(tFilename)
	})
	t.Run("initSuperblock", func(t *testing.T) {
		// Setup
		d, _ := createDisk(tFilename, tBlockCt)
		// Test
		if err := d.initSuperblock(); err != nil {
			t.Error(err)
		}
		block := make([]byte, BlockSize)
		var offset int64 = 0
		n, err := d.fd.ReadAt(block, offset)
		if err != nil || n == 0 {
			t.Errorf("Error while reading superblock: %v bytes read, %s", n, err)
		}
		// Extract values, only test subset that were inserted differently
		sig := block[:SbSigSize]
		dataBlockCtBytes := block[SbDataBlockCtOffset:(SbDataBlockCtOffset + SbDataBlockCtSize)]
		fatBlockCtBytes := block[SbFatBlockCtOffset:(SbFatBlockCtOffset + SbFatBlockCtSize)]
		dataBlockCt := int(binary.LittleEndian.Uint16(dataBlockCtBytes))
		fatBlockCt := int(fatBlockCtBytes[0])
		// Compare to expected
		builder := strings.Builder{}
		builder.Write(sig)
		sigExp, sigGot := SbSig, builder.String()
		if 0 != strings.Compare(sigExp, sigGot) {
			t.Errorf("Expected signature %s, Got %s", sigExp, sigGot)
		}
		if dataBlockCt != tBlockCt {
			t.Errorf("Expected data block count %v, Got %v", tBlockCt, dataBlockCt)
		}
		fatBlockCtExp := int(math.Ceil( (2 * float64(d.dataBlockCt)) / BlockSize ))
		if fatBlockCt != fatBlockCtExp {
			t.Errorf("Expected fat block count %v, Got %v", fatBlockCtExp, fatBlockCt)
		}
		// Same method of assignment for all variables, so only test a couple
		if sigGot != d.sig {
			t.Errorf("Read signature doesn't match structure value: %s, %s", sigGot, d.sig)
		}
		if dataBlockCt != d.dataBlockCt {
			t.Errorf("Read data block count doesn't match structure value: %v, %v", dataBlockCt, d.dataBlockCt)
		}
		// Teardown
		d.fd.Close()
		os.Remove(tFilename)
	})
	t.Run("initFS", func(t *testing.T) {
		// Setup
		d, _ := createDisk(tFilename, tBlockCt)
		// Test
		if err := d.initFS(); err != nil {
			t.Error(err)
		}
		fatBlks := int(math.Ceil( (2 * float64(d.dataBlockCt))/BlockSize ))
		totBlks := 2 + fatBlks + tBlockCt
		fLenExp := int64(totBlks * BlockSize)
		fStat, _ := d.fd.Stat()
		fLenGot := fStat.Size()
		if fLenGot != fLenExp {
			t.Errorf("Expected disk size %v, Got %v", fLenExp, fLenGot)
		}
		// Teardown
		d.fd.Close()
		os.Remove(tFilename)
	})
	// Test
	d, err := New(tFilename, tBlockCt)
	if err != nil {
		t.Error(err)
	}
	// Teardown
	d.fd.Close()
	os.Remove(tFilename)
}

func TestMount(t *testing.T) {
	// Setup
	tFilename, tBlockCt := "test.disk", 64
	d, _ := New(tFilename, tBlockCt)
	d.fd.Close()
	// Internal tests
	t.Run("readSuperblock", func(t *testing.T) {
		// Setup
		fd, _ := os.Open(tFilename)
		d := Disk{fd: fd}
		// Test
		d.readSuperblock()
		sigExp := SbSig
		fatBlockCtExp := int(math.Ceil( (2 * float64(d.dataBlockCt))/BlockSize ))
		blockCtExp := 2 + fatBlockCtExp + tBlockCt
		rootDirIndExp := 1 + fatBlockCtExp
		dataStartIndExp := 1 + rootDirIndExp
		dataBlockCtExp := tBlockCt
		if d.sig != sigExp {
			t.Errorf("Expected sig %s, Got %s", sigExp, d.sig)
		}
		if d.fatBlockCt != fatBlockCtExp {
			t.Errorf("Expected fat block ct %v, Got %v", dataBlockCtExp, d.dataBlockCt)
		}
		if d.blockCt != blockCtExp {
			t.Errorf("Expected block ct %v, Got %v", blockCtExp, d.blockCt)
		}
		if d.rootDirInd != rootDirIndExp {
			t.Errorf("Expected root dir index %v, Got %v", rootDirIndExp, d.rootDirInd)
		}
		if d.dataStartInd != dataStartIndExp {
			t.Errorf("Expected data start index %v, Got %v", dataStartIndExp, d.dataStartInd)
		}
		if d.dataBlockCt != dataBlockCtExp {
			t.Errorf("Expected data block ct %v, Got %v", dataBlockCtExp, d.dataBlockCt)
		}
		// Teardown
		fd.Close()
	})
	// Test
	disk, err := Mount(tFilename)
	if err != nil {
		t.Error(err)
	}
	if disk.fd == nil {
		t.Errorf("Nil file descriptor for '%s'", tFilename)
	}
	//Teardown
	disk.fd.Close()
	os.Remove(tFilename)
}
