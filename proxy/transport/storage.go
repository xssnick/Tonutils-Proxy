package transport

import (
	"fmt"
	"github.com/xssnick/tonutils-go/adnl/keys"
	"github.com/xssnick/tonutils-go/tl"
	"github.com/xssnick/tonutils-storage/storage"
	"sync"
)

type VirtualStorage struct {
	torrents map[string]*storage.Torrent
	mx       sync.RWMutex
}

func (v *VirtualStorage) VerifyOnStartup() bool {
	return false
}

func (v *VirtualStorage) GetForcedPieceSize() uint32 {
	//TODO implement me
	return 0
}

func NewVirtualStorage() *VirtualStorage {
	return &VirtualStorage{torrents: map[string]*storage.Torrent{}}
}

func (v *VirtualStorage) GetFS() storage.FS {
	panic("virtual")
}

func (v *VirtualStorage) GetAll() []*storage.Torrent {
	v.mx.RLock()
	defer v.mx.RUnlock()

	var res []*storage.Torrent
	for _, t := range v.torrents {
		res = append(res, t)
	}

	return res
}

func (v *VirtualStorage) GetTorrentByOverlay(overlay []byte) *storage.Torrent {
	v.mx.RLock()
	defer v.mx.RUnlock()

	return v.torrents[string(overlay)]
}

func (v *VirtualStorage) SetTorrent(t *storage.Torrent) error {
	id, err := tl.Hash(keys.PublicKeyOverlay{Key: t.BagID})
	if err != nil {
		return err
	}

	v.mx.Lock()
	defer v.mx.Unlock()

	v.torrents[string(id)] = t
	return nil
}

func (v *VirtualStorage) SetActiveFiles(bagId []byte, ids []uint32) error {
	panic("virtual")
}

func (v *VirtualStorage) GetActiveFiles(bagId []byte) ([]uint32, error) {
	panic("virtual")
}

func (v *VirtualStorage) GetPiece(bagId []byte, id uint32) (*storage.PieceInfo, error) {
	return nil, fmt.Errorf("virtual storage")
}

func (v *VirtualStorage) RemovePiece(bagId []byte, id uint32) error {
	return nil
}

func (v *VirtualStorage) SetPiece(bagId []byte, id uint32, p *storage.PieceInfo) error {
	return nil
}

func (v *VirtualStorage) PiecesMask(bagId []byte, num uint32) []byte {
	add := uint32(0)
	if num%8 != 0 {
		add++
	}
	return make([]byte, num/8+add)
}

func (v *VirtualStorage) UpdateUploadStats(bagId []byte, val uint64) error {
	return nil
}
