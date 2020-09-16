package disk

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	SbSig                = "NEWFATFS"
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
	message string
}

func (e CustomError) Error() string {
	return e.message
}

type InvalidFilenameError struct {
	filename string
}

func (e InvalidFilenameError) Error() string {
	return fmt.Sprintf("File Not Found: %s", e.filename)
}

type Disk struct {
	fd           *os.File // file descriptor for disk file
	sig          string   // filesystem signature
	blockCt      int      // total disk blocks
	rootDirInd   int      // block index of the root directory
	dataStartInd int      // disk block index of first data block
	dataBlockCt  int      // number of data blocks on disk
	fatBlockCt   int      // number of blocks used to store FAT
}

// Makes a new disk and initializes its filesystem
// Scope: exported
func New(filename string, dataBlocks int) (Disk, error) {
	d, err := createDisk(filename, dataBlocks)
	if err != nil {
		return d, err
	}

	if err = d.initFS(); err != nil {
		return Disk{}, err
	}

	return d, nil
}

// Loads a disk file and returns the associated structure
// Scope: exported
func Mount(filename string) (Disk, error) {
	if len(filename) == 0 {
		return Disk{}, InvalidFilenameError{filename}
	}
	// Open disk file
	fd, err := os.Open(filename)
	if err != nil {
		fd.Close()
		return Disk{}, err
	}
	// Create struct and read data from file
	d := Disk{fd: fd}
	err = d.readSuperblock()
	if err != nil {
		fd.Close()
		return Disk{}, err
	}
	return d, nil
}

// Instantiates a new disk and creates the associated file
// Scope: internal
func createDisk(filename string, dataBlocks int) (Disk, error) {
	if len(filename) == 0 {
		return Disk{}, InvalidFilenameError{filename}
	}

	file, err := os.Create(filename)
	if err != nil {
		os.Remove(filename)
		return Disk{}, err
	}

	return Disk{fd: file, dataBlockCt: dataBlocks}, nil
}

// Initializes the filesystem
// Scope: internal
func (d *Disk) initFS() error {
	numFATBlks := int(math.Ceil((2 * float64(d.dataBlockCt)) / BlockSize))
	numTotalBlks := 2 + numFATBlks + d.dataBlockCt
	// initialize full disk
	_, err := d.fd.Write(make([]byte, numTotalBlks*BlockSize))
	if err != nil {
		return err
	}
	// create superblock
	if err := d.initSuperblock(); err != nil {
		return err
	}
	return nil
}

// Initializes the superblock, called by initFS()
// Scope: internal
func (d *Disk) initSuperblock() error {
	// (2 bytes per FAT Entry) * (Num FAT Entries) / (Num bytes per block)
	numFatBlks := int(math.Ceil((2 * float64(d.dataBlockCt)) / BlockSize))
	// 1 block for superblock + 1 block for root directory + FAT + data
	numBlks := 2 + numFatBlks + d.dataBlockCt
	// initialize superblock byte slice and extract subslices for each section
	superblock := make([]byte, BlockSize)
	sig := superblock[:SbSigSize]
	blockCt := superblock[SbBlockCtOffset:(SbBlockCtOffset + SbBlockCtSize)]
	rootDirInd := superblock[SbRootDirIndOffset:(SbRootDirIndOffset + SbRootDirIndSize)]
	dataStartInd := superblock[SbDataStartIndOffset:(SbDataStartIndOffset + SbDataStartIndSize)]
	dataBlockCt := superblock[SbDataBlockCtOffset:(SbDataBlockCtOffset + SbDataBlockCtSize)]
	fatBlockCt := superblock[SbFatBlockCtOffset:(SbFatBlockCtOffset + SbFatBlockCtSize)]
	// calculate values and store in disk structure
	d.sig = SbSig
	d.blockCt = numBlks
	d.rootDirInd = 1 + numFatBlks
	d.dataStartInd = 2 + numFatBlks
	d.fatBlockCt = numFatBlks
	// write data to each subslice
	copy(sig, d.sig)
	binary.LittleEndian.PutUint16(blockCt, uint16(d.blockCt))
	binary.LittleEndian.PutUint16(rootDirInd, uint16(d.rootDirInd))
	binary.LittleEndian.PutUint16(dataStartInd, uint16(d.dataStartInd))
	binary.LittleEndian.PutUint16(dataBlockCt, uint16(d.dataBlockCt))
	fatBlockCt[0] = byte(d.fatBlockCt)
	// write byte slice to beginning of disk file
	var offset int64 = 0
	_, err := d.fd.WriteAt(superblock, offset)
	if err != nil {
		return err
	}
	return nil
}

func (d *Disk) readSuperblock() error {
	var offset int64 = 0
	superblock := make([]byte, BlockSize)
	_, err := d.fd.ReadAt(superblock, offset)
	if err != nil {
		return err
	}
	// load fields as subslices
	sig := superblock[:SbSigSize]
	blockCt := superblock[SbBlockCtOffset:(SbBlockCtOffset + SbBlockCtSize)]
	rootDirInd := superblock[SbRootDirIndOffset:(SbRootDirIndOffset + SbRootDirIndSize)]
	dataStartInd := superblock[SbDataStartIndOffset:(SbDataStartIndOffset + SbDataStartIndSize)]
	dataBlockCt := superblock[SbDataBlockCtOffset:(SbDataBlockCtOffset + SbDataBlockCtSize)]
	fatBlockCt := superblock[SbFatBlockCtOffset:(SbFatBlockCtOffset + SbFatBlockCtSize)]
	// read data from each subslice into correspond struct member
	builder := strings.Builder{}
	builder.Write(sig)
	d.sig = builder.String()
	d.blockCt = int(binary.LittleEndian.Uint16(blockCt))
	d.rootDirInd = int(binary.LittleEndian.Uint16(rootDirInd))
	d.dataStartInd = int(binary.LittleEndian.Uint16(dataStartInd))
	d.dataBlockCt = int(binary.LittleEndian.Uint16(dataBlockCt))
	d.fatBlockCt = int(fatBlockCt[0])

	return nil
}
