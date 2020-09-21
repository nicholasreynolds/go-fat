package disk

import "fmt"

type CustomError struct {
	message string
}

type MemberUndefinedError struct {
	name string
}

type InvalidFilenameError struct {
	filename string
}

type FileAlreadyInUseError struct {
	filename string
}

type FileAlreadyExistsError struct {
	filename string
}

type FileNotFoundError struct {
	filename string
}

type FileNotOpenError struct {
	filename string
}

type FullDiskError struct {}
type RootDirFullError struct {}

func (e CustomError) Error() string {
	return e.message
}

func (e MemberUndefinedError) Error() string {
	return fmt.Sprintf("Member undefined: %s", e.name)
}

func (e InvalidFilenameError) Error() string {
	return fmt.Sprintf("File Not Found: %s", e.filename)
}

func (e FileAlreadyInUseError) Error() string {
	return fmt.Sprintf("File already in use: %s", e.filename)
}

func (e FileAlreadyExistsError) Error() string {
	return fmt.Sprintf("File already exists: %s", e.filename)
}

func (e FileNotFoundError) Error() string {
	return fmt.Sprintf("File not found: %s", e.filename)
}

func (e FileNotOpenError) Error() string {
	return fmt.Sprintf("File not open: %s", e.filename)
}

func (e FullDiskError) Error() string {
	return "Disk is full, no data blocks available for writing"
}

func (e RootDirFullError) Error() string {
	return "Root directory full, max file limit reached"
}