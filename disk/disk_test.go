package disk

import (
	"encoding/binary"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDisk_New(t *testing.T) {
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
		fatBlockCtExp := int(math.Ceil((FatEntrySize * float64(d.dataBlockCt)) / BlockSize))
		if fatBlockCt != fatBlockCtExp {
			t.Errorf("Expected fat block count %v, Got %v", fatBlockCtExp, fatBlockCt)
		}
		// Same method of assignment for all variables, so only test a couple
		if 0 != strings.Compare(sigGot, d.sig) {
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
		fatBlks := int(math.Ceil((FatEntrySize * float64(d.dataBlockCt)) / BlockSize))
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

func TestDisk_Mount(t *testing.T) {
	// Setup
	tFilename, tBlockCt := "test.disk", 64
	d, _ := New(tFilename, tBlockCt)
	d.fd.Close()
	// Internal tests
	t.Run("readSuperblock", func(t *testing.T) {
		// Setup
		fd, _ := os.Open(tFilename)
		d := Disk{fd: fd, dataBlockCt: tBlockCt}
		d.initFS()
		// Test
		d.readSuperblock()
		sigExp := SbSig
		fatBlockCtExp := int(math.Ceil((FatEntrySize * float64(d.dataBlockCt)) / BlockSize))
		blockCtExp := 2 + fatBlockCtExp + tBlockCt
		rootDirIndExp := 1 + fatBlockCtExp
		dataStartIndExp := 1 + rootDirIndExp
		dataBlockCtExp := tBlockCt
		if 0 != strings.Compare(d.sig, sigExp) {
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

func TestDisk_Create(t *testing.T) {
	// Setup
	tDiskFilename, tBlockCt := "test.disk", 64
	tFilename := "test.txt"
	// Test
	t.Run("initFatChain", func(t *testing.T) {
		// Setup
		d, _ := New(tDiskFilename, tBlockCt)
		// Test
		blockInd, err := d.initFatChain()
		if err != nil {
			t.Error(err)
		}
		fatInd := FatEntrySize * blockInd
		fatBuff := make([]byte, d.fatBlockCt*BlockSize)
		d.fd.ReadAt(fatBuff, BlockSize) // fat is next block after superblock
		eocGot := binary.LittleEndian.Uint16(fatBuff[fatInd : fatInd+FatEntrySize])
		if eocGot != FatEoc {
			t.Errorf("Expected EOC value %v, Got %v", FatEoc, eocGot)
		}
		// Teardown
		d.fd.Close()
		os.Remove(tDiskFilename)
	})
	t.Run("initRootEntry", func(t *testing.T) {
		// Setup
		d, _ := New(tDiskFilename, tBlockCt)
		blockInd, err := d.initFatChain()
		if err != nil {
			t.Error(err)
		}
		// Test
		var entryInd int
		entryInd, err = d.initRootEntry(tFilename, blockInd)
		if err != nil {
			t.Error(err)
		}
		rootBuff := make([]byte, BlockSize)
		offset := int64(d.rootDirInd * BlockSize)
		d.fd.ReadAt(rootBuff, offset)
		entryPos := entryInd * RootEntrySize
		rootEntry := rootBuff[entryPos : entryPos+RootEntrySize]
		builder := strings.Builder{}
		builder.Write(rootEntry[:RootEntryFilenameSize])
		fnGot := strings.Trim(builder.String(), "\x00")
		if 0 != strings.Compare(fnGot, tFilename) {
			t.Errorf("Expected filename %s, Got %s", tFilename, fnGot)
		}
		szGot := binary.
			LittleEndian.
			Uint32(rootEntry[RootEntryFilenameSize : RootEntryFilenameSize+RootEntrySizeFieldSize])
		if szGot != 0 {
			t.Errorf("Expected size 0 bytes, Got %v", szGot)
		}
		stBlkOffset := RootEntryFilenameSize + RootEntrySizeFieldSize
		startBlkGot := binary.
			LittleEndian.
			Uint16(rootEntry[stBlkOffset : stBlkOffset+RootEntryStartBlockSize])
		if startBlkGot != 0 {
			t.Errorf("Expected start block index 0, Got %v", startBlkGot)
		}
		// Teardown
		d.fd.Close()
		os.Remove(tDiskFilename)
	})
	d, _ := New(tDiskFilename, tBlockCt)
	file, err := d.Create(tFilename)
	if err != nil {
		t.Error(err)
	}
	if file.name != tFilename {
		t.Errorf("Expected filename %s, Got %s", tFilename, file.name)
	}
	if !reflect.DeepEqual(*(file.disk), d) {
		t.Errorf("File disk reference mismatch")
	}
	if file.desc != 0 {
		t.Errorf("Expected file desc 0, Got %v", file.desc)
	}
	if file.offset != 0 {
		t.Errorf("Expected file offset 0, Got %v", file.offset)
	}
	if file.size != 0 {
		t.Errorf("Expected file size 0, Got %v", file.size)
	}
}

func TestDisk_Open(t *testing.T) {
	// Setup
	tDiskFilename, tBlockCt := "test.disk", 64
	tFilename := "test.txt"
	// Test
	t.Run("checkIsOpen", func(t *testing.T) {
		// Setup
		d, _ := New(tDiskFilename, tBlockCt)
		// Test
		// check ret false for nonexistent file
		isOpen := d.checkIsOpen(tFilename)
		if isOpen {
			t.Error("Expected open flag false, Got true")
		}
		// check ret true once flag set
		d.Create(tFilename)
		isOpen = d.checkIsOpen(tFilename)
		if !isOpen {
			t.Error("Expected open flag true, Got false")
		}
		// Teardown
		d.fd.Close()
		os.Remove(tDiskFilename)
	})
	t.Run("loadRootEntry", func(t *testing.T) {
		// Setup
		d, _ := New(tDiskFilename, tBlockCt)
		fExp, _ := d.Create(tFilename)
		// Test
		fGot := &File{name: tFilename}
		err := d.loadRootEntry(fGot)
		if err != nil {
			t.Error(err)
		}
		if 0 != strings.Compare(fGot.name, fExp.name) {
			t.Errorf("Expected filename %s, Got %s", fExp.name, fGot.name)
		}
		if fGot.size != fExp.size {
			t.Errorf("Expected file size %v, Got %v", fExp.size, fGot.size)
		}
		if fGot.desc != fExp.desc {
			t.Errorf("Expected start block index %v, Got %v", fExp.desc, fGot.desc)
		}
		// Teardown
		d.fd.Close()
		os.Remove(tDiskFilename)
	})
	d, _ := New(tDiskFilename, tBlockCt)
	file, _ := d.Create(tFilename)
	err := file.Close()
	if err != nil {
		t.Error(err)
	}
	file, err = d.Open(tFilename)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(*(file.disk), d) {
		t.Error("File disk reference mismatch")
	}
	if file.offset != 0 {
		t.Errorf("Expected file offset 0, Got %v", file.offset)
	}
}
