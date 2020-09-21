package disk

type File struct {
	name   string // filename
	disk   *Disk  // disk reference. Necessary for read/write ops
	desc   int    // file descriptor i.e. the block index on disk
	offset int    // byte offset from
	size   int    // size in bytes
}

func (f *File) Write(data []byte) (int, error) {
	return 0, nil
}

func (f *File) WriteAt(data []byte, offset int) (int, error) {
	return 0, nil
}

func (f *File) Read(buff []byte) (int, error) {
	return 0, nil
}

func (f *File) ReadAt(buff []byte, offset int) (int, error) {
	return 0, nil
}

func (f *File) Close() error {
	if f == nil {
		return CustomError{"Nil Structure"}
	}
	if len(f.name) == 0 {
		return MemberUndefinedError{"name"}
	}
	if _, ok := f.disk.open[f.name]; !ok {
		return FileNotOpenError{f.name}
	}
	delete(f.disk.open, f.name)
	return nil
}
