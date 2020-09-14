package disk

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

func TestCreate(t *testing.T) {
	// Setup
	filename, blockCt := "test.disk", 64
	// Internal tests
	t.Run("newDisk", func(t *testing.T) {
		d, err := newDisk(filename, blockCt)
		if err != nil {
			// Covers any file-related kernel and i/o errors
			t.Error(err)
		}
		if d.fd == nil {
			t.Errorf("Nil file descriptor for '%s'", filename)
		}
		if d.dataBlks != blockCt {
			t.Errorf("Expected %v data blocks, Got %v", blockCt, d.dataBlks)
		}
		//Teardown
		os.Remove(filename)
	})
	t.Run("initSuperblock", func(t *testing.T) {
		// Setup
		d, _ := newDisk(filename, blockCt)
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
		sigExp, sigGot := "NEWFATFS", builder.String()
		if 0 != strings.Compare(sigExp, sigGot) {
			t.Errorf("Expected signature %s, Got %s", sigExp, sigGot)
		}
		if dataBlockCt != d.dataBlks {
			t.Errorf("Expected data block count %v, Got %v", d.dataBlks, dataBlockCt)
		}
		fatBlockCtExp := int(math.Ceil( (2 * float64(d.dataBlks)) / BlockSize ))
		if fatBlockCt != fatBlockCtExp {
			t.Errorf("Expected fat block count %v, Got %v", fatBlockCtExp, fatBlockCt)
		}
		// Teardown
		os.Remove(filename)
	})
	t.Run("initFS", func(t *testing.T) {
		// Setup
		d, _ := newDisk(filename, blockCt)
		// Test
		if err := d.initFS(); err != nil {
			t.Error(err)
		}
		// (2 blocks for sb and rootdir) * (number of bytes per block) +
		// (2 bytes for each fat entry) * (num fat entries i.e. data block count) +
		// (number of data blocks) * (number of bytes per block
		fLenExp := int64(2*BlockSize + 2*blockCt + blockCt*BlockSize)
		fStat, _ := d.fd.Stat()
		fLenGot := fStat.Size()
		if fLenGot != fLenExp {
			t.Errorf("Expected disk size %v, Got %v", fLenExp, fLenGot)
		}
		// Teardown
		os.Remove(filename)
	})
	// Test
	_, err := Create(filename, blockCt)
	if err != nil {
		t.Error(err)
	}
	// Teardown
	os.Remove(filename)
}
