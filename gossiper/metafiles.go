package gossiper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/eliasmpw/Peerster/common"
	"os"
	"path/filepath"
	"sync"
)

type FileMetaData struct {
	Origin    string
	Name      string
	Size      uint
	MetaFile  []byte
	HashValue []byte
}

// Get the hash of a chunk in position i
func (fmd FileMetaData) GetChunkHash(i int) []byte {
	return fmd.ChunkHashes()[i]
}

// Get a slice with all hashes of the chunks
func (fmd FileMetaData) ChunkHashes() [][]byte {
	hashSizeInBytes := int(myGossiper.hashSize) / 8
	arrayHashes := make([][]byte, 0)
	for i := 0; i < len(fmd.MetaFile); i += hashSizeInBytes {
		hash := fmd.MetaFile[i : i+hashSizeInBytes]
		aux := make([]byte, len(hash))
		copy(aux, hash)
		arrayHashes = append(arrayHashes, aux)
	}
	return arrayHashes
}

// Return the index position of a chunk
func (fmd FileMetaData) GetPositionOfChunk(chunkHash []byte) *int {
	allHashes := fmd.ChunkHashes()
	for i := 0; i < len(allHashes); i++ {
		if bytes.Equal(chunkHash, allHashes[i]) {
			return &i
		}
	}
	return nil
}

func GetChunkNumber(metaFile []byte) uint {
	dataLen := uint(len(metaFile))
	hashSizeInBytes := myGossiper.hashSize / 8
	return dataLen / hashSizeInBytes
}

// Structure containing slice of meta data files and a mutex
type MetaDataList struct {
	metaDataFiles []FileMetaData
	mutex         *sync.Mutex
}

func NewMetaDataList() MetaDataList {
	return MetaDataList{
		metaDataFiles: make([]FileMetaData, 0),
		mutex:         &sync.Mutex{},
	}
}

// Check if a FileMetaData entry already exists
func (mdl *MetaDataList) Exists(fmd FileMetaData) bool {
	mdl.mutex.Lock()
	for _, m := range mdl.metaDataFiles {
		if bytes.Equal(fmd.HashValue, m.HashValue) {
			mdl.mutex.Unlock()
			return true
		}
	}
	mdl.mutex.Unlock()
	return false
}

// Insert a FileMetaData if it doesn't exist yet in our list
func (mdl *MetaDataList) Add(fmd FileMetaData) {
	if mdl.Exists(fmd) {
		return
	}
	mdl.mutex.Lock()
	mdl.metaDataFiles = append(mdl.metaDataFiles, fmd)
	mdl.mutex.Unlock()
}

// Get the FileMetaData by file hash
func (mdl *MetaDataList) GetByHash(hash []byte) *FileMetaData {
	mdl.mutex.Lock()
	for _, fmd := range mdl.metaDataFiles {
		if bytes.Equal(fmd.HashValue, hash) {
			mdl.mutex.Unlock()
			return &fmd
		}
	}
	mdl.mutex.Unlock()
	return nil
}

// Helper functions

// Split a byte slice of a file to chunks
func SplitToChunks(data []byte, chunkSize uint) *[][]byte {
	length := len(data)
	remaining := uint(length)

	chunksSlice := make([][]byte, 0)

	for i := uint(0); remaining > 0; i += chunkSize {
		j := chunkSize
		if j > remaining {
			j = remaining
		}

		chunksSlice = append(chunksSlice, data[i:i+j])
		remaining -= j
	}

	return &chunksSlice
}

// Create hash for every chunk in a slice
func CreateChunkHashes(chunks *[][]byte) [][]byte {
	chunkHashes := make([][]byte, 0)

	for i := 0; i < len(*chunks); i++ {
		h := sha256.New()
		h.Write((*chunks)[i])
		chunkHashes = append(chunkHashes, h.Sum(nil))
	}

	return chunkHashes
}

// Write file bytes on disk
func WriteFileOnDisk(data []byte, dir, fileName string) {
	os.MkdirAll(dir, os.ModePerm)
	fullPath := dir + fileName
	file, err := os.Create(fullPath)
	common.CheckError(err)
	defer file.Close()
	_, err = file.Write(data)
	file.Sync()
}

func ChunkFileName(chunk []byte) string {
	auxHash := sha256.New()
	auxHash.Write(chunk)
	hashString := hex.EncodeToString(auxHash.Sum(nil))
	// Limit chunk filename size
	if len(hashString) > 249 {
		hashString = hashString[:249]
	}
	filename := hashString + ".chunk"
	return filename
}

func WriteChunksOnDisk(chunks [][]byte, dir, fileName string) {
	absPath, err := filepath.Abs("")
	common.CheckError(err)
	for _, chunk := range chunks {
		// storing chunk i
		filename := ChunkFileName(chunk)
		chunkDir := absPath + string(os.PathSeparator) + dir
		WriteFileOnDisk(chunk, chunkDir, filename)
	}
}

func ReconstructFromChunks(chunks *[][]byte) *[]byte {
	completeFile := make([]byte, 0)
	for i := 0; i < len(*chunks); i++ {
		completeFile = append(completeFile, (*chunks)[i]...)
	}

	return &completeFile
}

func GetChunkFilename(hash []byte) string {
	offset := myGossiper.hashSize / 8

	hashString := hex.EncodeToString(hash[:offset])
	// Limit chunk filename size
	if len(hashString) > 249 {
		hashString = hashString[:249]
	}
	filename := hashString + ".chunk"
	return filename
}
