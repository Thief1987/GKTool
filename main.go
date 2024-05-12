package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	arc                  *os.File
	name_buf             bytes.Buffer
	foldersCount_massive = make([]uint32, 20)
	filesCount_massive   = make([]uint32, 20)
	path_level           int
	filepath_massive     = make([]string, 20)
	path                 string
	savepos_massive      = make([]int64, 20)
	filecount            int
)

func ReadUint32(r io.Reader) uint32 {
	var buf bytes.Buffer
	io.CopyN(&buf, r, 4)
	return binary.LittleEndian.Uint32(buf.Bytes())
}

func unpack() {
	for j := 0; j <= path_level; j++ {
		path = path + filepath_massive[j] + "\\"
	}
	path = strings.Replace(path, "\\", "", 1)
	current_path := path
	os.MkdirAll(path, 0700)
	path = ""
	for i := 0; i < int(foldersCount_massive[path_level]); i++ {
		io.CopyN(&name_buf, arc, 0x100)
		FolderName := strings.Replace(name_buf.String(), "\x00", "", -1)
		path_info_offset := ReadUint32(arc)
		path_level++
		savepos_massive[path_level], _ = arc.Seek(0, 1)
		arc.Seek(int64(path_info_offset), 0)
		filepath_massive[path_level] = FolderName
		foldersCount_massive[path_level] = ReadUint32(arc)
		filesCount_massive[path_level] = ReadUint32(arc)
		name_buf.Reset()
		unpack()
	}
	for j := 0; j < int(filesCount_massive[path_level]); j++ {
		var buf bytes.Buffer
		io.CopyN(&name_buf, arc, 0x100)
		FileName := strings.Replace(name_buf.String(), "\x00", "", -1)
		FileSize := ReadUint32(arc)
		BlockSize := ReadUint32(arc)
		foffset := ReadUint32(arc)
		savepos, _ := arc.Seek(0, 1)
		file, _ := os.Create(current_path + FileName)
		arc.Seek(int64(foffset), 0)
		if int(FileSize) == int(BlockSize) {
			io.CopyN(file, arc, int64(FileSize))
		} else {
			_ = ReadUint32(arc) //FileSize... again
			FileZsize := ReadUint32(arc)
			io.CopyN(&buf, arc, int64(FileZsize))
			r, _ := zlib.NewReader(&buf)
			io.Copy(file, r)
		}
		fmt.Printf("0x%X       %v        %s\n", foffset, FileSize, current_path+FileName)
		arc.Seek(savepos, 0)
		filecount = filecount + 1
		name_buf.Reset()
	}
	arc.Seek(savepos_massive[path_level], 0)
	path_level--
}

//TODO repack

func main() {
	args := os.Args
	arcName := args[1]
	arc, _ = os.Open(arcName)
	defer arc.Close()
	io.CopyN(&name_buf, arc, 0x100)
	RootFolderName := strings.Replace(name_buf.String(), "\x00", "", -1)
	fmt.Println(name_buf.String())
	os.MkdirAll(RootFolderName, 0700)
	os.Chdir(RootFolderName)
	name_buf.Reset()
	foldersCount_massive[0] = ReadUint32(arc) //RootFolderDirCount
	filesCount_massive[0] = ReadUint32(arc)   // RootFolderFileCount
	_ = ReadUint32(arc)                       //DataOffset
	fmt.Println("Offset       Size                     Name   ")
	unpack()
	fmt.Printf("\nSuccesfully extracted %v files\n", filecount)
}
