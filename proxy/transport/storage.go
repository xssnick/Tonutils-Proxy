package transport

import (
	"fmt"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-storage/storage"
	"sync"
)

type VirtualStorage struct {
	torrents map[string]*storage.Torrent
	mx       sync.Mutex
}

func NewVirtualStorage() *VirtualStorage {
	return &VirtualStorage{torrents: map[string]*storage.Torrent{}}
}

func (v *VirtualStorage) GetFS() storage.FS {
	panic("virtual")
}

func (v *VirtualStorage) GetAll() []*storage.Torrent {
	panic("virtual")
}

func (v *VirtualStorage) GetTorrentByOverlay(overlay []byte) *storage.Torrent {
	v.mx.Lock()
	defer v.mx.Unlock()

	return v.torrents[string(overlay)]
}

func (v *VirtualStorage) SetTorrent(t *storage.Torrent) error {
	id, err := adnl.ToKeyID(adnl.PublicKeyOverlay{Key: t.BagID})
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
