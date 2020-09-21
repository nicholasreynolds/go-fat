package disk

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
)

const (
	SbSig                   = "NEWFATFS"
	BlockSize               = 4096
	SbSigSize               = 8
	SbBlockCtOffset         = 0x08
	SbBlockCtSize           = 2
	SbRootDirIndOffset      = 0x0A
	SbRootDirIndSize        = 2
	SbDataStartIndOffset    = 0x0C
	SbDataStartIndSize      = 2
	SbDataBlockCtOffset     = 0x0E
	SbDataBlockCtSize       = 2
	SbFatBlockCtOffset      = 0x10
	SbFatBlockCtSize        = 1
	SbPaddSize              = 4079
	SbPaddOffset            = 0x11
	FatEoc                  = 0xFFFF
	FatEntrySize            = 2
	FatEntryUnused          = 0
	RootEntrySize           = 32
	RootEntryFilenameSize   = 16
	RootEntrySizeFieldSize  = 4
	RootEntryStartBlockSize = 2
)

type Disk struct {
	fd           *os.File // file descriptor for disk file
	sig          string   // filesystem signature
	blockCt      int      // total disk blocks
	rootDirInd   int      // block index of the root directory
	dataStartInd int      // disk block index of first data block
	dataBlockCt  int      // number of data blocks on disk
	fatBlockCt   int      // number of blocks used to store FAT
	open		 map[string]bool // map of all open files
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

func (d *Disk) Create(filename string) (File, error) {
	// find free data block entry in fat
	blockInd, err := d.initFatChain()
	if err != nil {
		return File{}, err
	}
	// add root directory entry for file
	rootInd, err := d.initRootEntry(filename, blockInd)
	if err != nil {
		return File{}, err
	}
	// set file open flag true
	d.open[filename] = true
	return File{
		name:   filename,
		disk:   d,
		desc:   rootInd,
		offset: 0,
		size:   0,
	}, nil
}

// Opens the file with given filename, if not already open.
// Returns: (File structure reference, any error that occurred)
func (d *Disk) Open(filename string) (File, error) {
	if d.checkIsOpen(filename) {
		return File{}, FileAlreadyInUseError{filename}
	}
	file := File{
		name:   filename,
		disk:   d,
		desc:   0,
		offset: 0,
		size:   0,
	}
	// load root entry values into file struct
	err := d.loadRootEntry(&file)
	if err != nil {
		return File{}, err
	}
	// if no errors encountered, set open flag true
	d.open[filename] = true
	return file, nil
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

	return Disk{
		fd: file,
		dataBlockCt: dataBlocks,
		open: make(map[string]bool),
	}, nil
}

// Initializes the filesystem
// Scope: internal
func (d *Disk) initFS() error {
	numFATBlks := int(math.Ceil((FatEntrySize * float64(d.dataBlockCt)) / BlockSize))
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
	numFatBlks := int(math.Ceil((FatEntrySize * float64(d.dataBlockCt)) / BlockSize))
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

// Locates a free fat entry and writes End-Of-Chain value to it.
// Otherwise returns a Full Disk Error
func (d *Disk) initFatChain() (int, error) {
	fatBuff := make([]byte, d.fatBlockCt*BlockSize)
	offset := int64(BlockSize)
	d.fd.ReadAt(fatBuff, offset)
	for i := 0; i < len(fatBuff); i += FatEntrySize {
		fatEntry := fatBuff[i : i+FatEntrySize]
		fatVal := binary.LittleEndian.Uint16(fatEntry)
		// find unused fat entry (i.e. has value 0)
		if fatVal == FatEntryUnused {
			binary.LittleEndian.PutUint16(fatEntry, FatEoc)
			d.fd.WriteAt(fatBuff, offset)
			return i, nil
		}
	}
	return 0, FullDiskError{}
}

// Writes a new root directory entry for the specified file, if space is available
// Returns: (index of entry in directory, any error encountered)
// Scope: Internal
func (d *Disk) initRootEntry(filename string, startBlock int) (int, error) {
	rootBuff := make([]byte, BlockSize)
	offset := int64(d.rootDirInd * BlockSize)
	d.fd.ReadAt(rootBuff, offset)
	for i := 0; i < len(rootBuff); i += RootEntrySize {
		rootEntry := rootBuff[i : i+RootEntrySize]
		name := rootEntry[:RootEntryFilenameSize]
		// check if entry is empty (i.e. name is null)
		if name[0] == 0 {
			// set filename
			copy(name, filename)
			// set first data block
			dtBlkOffset := RootEntryFilenameSize + RootEntrySizeFieldSize
			first := rootEntry[dtBlkOffset : dtBlkOffset+RootEntryStartBlockSize]
			binary.LittleEndian.PutUint16(first, uint16(startBlock))
			// write back to disk
			d.fd.WriteAt(rootBuff, offset)
			return i, nil
		}
		// check if filename already exists
		builder := strings.Builder{}
		builder.Write(name)
		if strings.Compare(builder.String(), filename) == 0 {
			return 0, FileAlreadyExistsError{filename}
		}
	}
	return 0, RootDirFullError{}
}

func (d *Disk) checkIsOpen(filename string) bool {
	// check filename is in map and open flag is set to true
	v, ok := d.open[filename]
	return ok && v
}

func (d *Disk) loadRootEntry(file *File) error {
	if file == nil {
		return CustomError{"File structure nil"}
	}
	if len(file.name) == 0 {
		return CustomError{"Filename empty"}
	}
	// extract root directory
	rootBuff := make([]byte, BlockSize)
	rootOffset := int64(d.rootDirInd*BlockSize)
	d.fd.ReadAt(rootBuff, rootOffset)
	// find root entry for filename and load values into struct
	for i := 0; i < len(rootBuff); i += RootEntrySize {
		entry := rootBuff[i : i+RootEntrySize]
		nameBuilder := strings.Builder{}
		nameBuilder.Write(entry[:RootEntryFilenameSize])
		// remove excess null characters
		name := strings.Trim(nameBuilder.String(), "\x00")
		// determine if current entry file name matches query
		if 0 == strings.Compare(name, file.name) {
			dtBlkOffset := RootEntryFilenameSize+RootEntrySizeFieldSize
			size := entry[RootEntryFilenameSize : dtBlkOffset]
			file.size = int(binary.LittleEndian.Uint32(size))
			dtBlk := entry[dtBlkOffset : dtBlkOffset+RootEntryStartBlockSize]
			file.desc = int(binary.LittleEndian.Uint16(dtBlk))
			return nil
		}
	}
	return FileNotFoundError{file.name}
}