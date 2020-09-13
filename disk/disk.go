package disk

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	BlockSize            = 4096
	SbSigSize            = 8
	SbBlockCtOffset      = 0x08
	SbBlockCtSize        = 2
	SbRootDirIndOffset   = 0x0A
	SbRootDirIndSize     = 2
	SbDataStartIndOffset = 0x0C
	SbDataStartIndSize   = 2
	SbDataBlockCtOffset  = 0x0E
	SbDataBlockCtSize    = 2
	SbFatBlockCtOffset   = 0x10
	SbFatBlockCtSize     = 1
	SbPaddSize           = 4079
	SbPaddOffset         = 0x11
)

type CustomError struct {
	message 	string
}

func (e CustomError) Error() string {
	return e.message
}

type InvalidFilenameError struct {
	filename	string
}

func (e InvalidFilenameError) Error() string {
	return fmt.Sprintf("File Not Found: %s", e.filename)
}

type Disk struct {
	fd       *os.File
	dataBlks int
}

// Creates the disk file and initializes its filesystem
func Create(filename string, dataBlocks int) (Disk, error) {
	d, err := newDisk(filename, dataBlocks)
	if err != nil {
		return d, err
	}

	if err = d.initFS(); err != nil {
		return Disk{}, err
	}

	return d, nil
}

// Instantiates a new disk and creates the associated file
func newDisk(filename string, dataBlocks int) (Disk, error) {
	if len(filename) == 0 {
		return Disk{}, InvalidFilenameError{filename}
	}

	file, err := os.Create(filename)
	if err != nil {
		return Disk{}, err
	}

	return Disk{file, dataBlocks}, nil
}

// Initializes the filesystem
func (d Disk) initFS() error {
	numFATBlks := (2 * d.dataBlks) / BlockSize
	numTotalBlks := 2 + numFATBlks + d.dataBlks
	// initalize full disk
	_, err := d.fd.Write(make([]byte, numTotalBlks))
	if err != nil {
		return err
	}
	// create superblock
	if err := d.initSuperblock(); err != nil {
		return err
	}
	return nil
}

// Initializes the superblock
func (d Disk) initSuperblock() error {
	// (2 bytes per FAT Entry) * (Num FAT Entries) / (Num bytes per block)
	numFatBlks := (2 * d.dataBlks) / BlockSize
	// 1 block for superblock + 1 block for root directory + FAT + data
	numBlks := 2 + numFatBlks + d.dataBlks
	// initialize superblock byte slice and extract subslices for each section
	superblock := make([]byte, BlockSize)
	sig := superblock[:SbSigSize]
	blockCt := superblock[SbBlockCtOffset:(SbBlockCtOffset + SbBlockCtSize)]
	rootDirInd := superblock[SbRootDirIndOffset:(SbRootDirIndOffset + SbRootDirIndSize)]
	dataStartInd := superblock[SbDataStartIndOffset:(SbDataStartIndOffset + SbDataStartIndSize)]
	dataBlockCt := superblock[SbDataBlockCtOffset:(SbDataBlockCtOffset + SbDataBlockCtSize)]
	fatBlockCt := superblock[SbFatBlockCtOffset:(SbFatBlockCtOffset + SbFatBlockCtSize)]
	// write data to each subslice
	copy(sig, "NEWFATFS")
	binary.LittleEndian.PutUint16(blockCt, uint16(numBlks))
	binary.LittleEndian.PutUint16(rootDirInd, uint16(1 + numFatBlks))
	binary.LittleEndian.PutUint16(dataStartInd, uint16(2 + numFatBlks))
	binary.LittleEndian.PutUint16(dataBlockCt, uint16(d.dataBlks))
	fatBlockCt[0] = byte(numFatBlks)
	// write byte slice to disk file
	_, err := d.fd.Write(superblock)
	if err != nil {
		return err
	}
	return nil
}
