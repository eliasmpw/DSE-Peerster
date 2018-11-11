package gossiper

import (
	"sync"
)

// Contains information related to a download that is in progress
type FileDownload struct {
	metaData  FileMetaData
	Chunks    [][]byte
	NextChunk uint
	LastChunk uint
}

// List of file downloads that are in progress
type FileDownloadsList struct {
	fileDownloads map[string]*FileDownload
	mutex         *sync.Mutex
}

func NewFileDownloadsList() *FileDownloadsList {
	return &FileDownloadsList{
		fileDownloads: make(map[string]*FileDownload),
		mutex:         &sync.Mutex{},
	}
}

// Get a FileDownload in the list by the hashValue
func (fdl *FileDownloadsList) GetByHash(hashValue []byte) *FileDownload {
	fdl.mutex.Lock()
	r := fdl.fileDownloads[string(hashValue)]
	fdl.mutex.Unlock()
	return r
}

// Return the chunk by hashValue in the FileDownloadsList list
func (fdl *FileDownloadsList) GetChunkByHash(hash []byte) *[]byte {
	fdl.mutex.Lock()
	for _, download := range fdl.fileDownloads {
		// Check if the chunk is inside this download
		index := download.metaData.GetPositionOfChunk(hash)
		if index != nil {
			// found the chunk
			chunk := download.Chunks[*index]
			fdl.mutex.Unlock()
			return &chunk
		}
	}
	fdl.mutex.Unlock()
	return nil
}

func (fdl *FileDownloadsList) Add(f *FileDownload) bool {
	fdl.mutex.Lock()
	if fdl.fileDownloads[string(f.metaData.HashValue)] != nil {
		// Already Exists
		return false
	}
	// Add to file downloads
	fdl.fileDownloads[string(f.metaData.HashValue)] = f
	fdl.mutex.Unlock()
	return true
}

func (fdl *FileDownloadsList) Remove(f *FileDownload) {
	fdl.mutex.Lock()
	fdl.fileDownloads[string(f.metaData.HashValue)] = nil
	fdl.mutex.Unlock()
}
