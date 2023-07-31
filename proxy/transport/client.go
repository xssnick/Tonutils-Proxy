package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-go/adnl/address"
	"github.com/xssnick/tonutils-go/adnl/rldp"
	rldphttp "github.com/xssnick/tonutils-go/adnl/rldp/http"
	"github.com/xssnick/tonutils-go/tl"
	"github.com/xssnick/tonutils-go/ton/dns"
	"github.com/xssnick/tonutils-storage/storage"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const _ChunkSize = 1 << 17
const _RLDPMaxAnswerSize = 2*_ChunkSize + 1024

type DHT interface {
	StoreAddress(ctx context.Context, addresses address.List, ttl time.Duration, ownerKey ed25519.PrivateKey, copies int) (int, []byte, error)
	FindAddresses(ctx context.Context, key []byte) (*address.List, ed25519.PublicKey, error)
	Close()
}

type Resolver interface {
	Resolve(ctx context.Context, domain string) (*dns.Domain, error)
}

type RLDP interface {
	Close()
	DoQuery(ctx context.Context, maxAnswerSize int64, query, result tl.Serializable) error
	SetOnQuery(handler func(transferId []byte, query *rldp.Query) error)
	SetOnDisconnect(handler func())
	SendAnswer(ctx context.Context, maxAnswerSize int64, queryId, transferId []byte, answer tl.Serializable) error
}

type ADNL interface {
	RemoteAddr() string
	GetID() []byte
	Query(ctx context.Context, req, result tl.Serializable) error
	SetDisconnectHandler(handler func(addr string, key ed25519.PublicKey))
	SetCustomMessageHandler(handler func(msg *adnl.MessageCustom) error)
	SendCustomMessage(ctx context.Context, req tl.Serializable) error
	Close()
}

type bagInfo struct {
	torrent    *storage.Torrent
	downloader storage.TorrentDownloader
}

var Connector = func(ctx context.Context, addr string, peerKey ed25519.PublicKey, ourKey ed25519.PrivateKey) (ADNL, error) {
	return adnl.Connect(ctx, addr, peerKey, ourKey)
}

var newRLDP = func(a ADNL) RLDP {
	return rldp.NewClient(a)
	// return rldp.NewClientV2(a) // for rldp2, but it is too early now
}

type siteInfo struct {
	Actor any

	LastUsed int64
	mx       sync.RWMutex
}

type rldpInfo struct {
	ActiveClient RLDP

	ID   ed25519.PublicKey
	Addr string
}

type Transport struct {
	dht              DHT
	resolver         Resolver
	storageConnector storage.NetConnector
	store            *VirtualStorage

	activeSites map[string]*siteInfo

	activeRequests map[string]*payloadStream
	globalCtx      context.Context
	stop           func()
	mx             sync.RWMutex
}

func NewTransport(dht DHT, resolver Resolver, storeConn storage.NetConnector, store *VirtualStorage) *Transport {
	t := &Transport{
		dht:              dht,
		resolver:         resolver,
		storageConnector: storeConn,
		store:            store,
		activeRequests:   map[string]*payloadStream{},
		activeSites:      map[string]*siteInfo{},
	}
	t.globalCtx, t.stop = context.WithCancel(context.Background())
	go t.cleaner()
	return t
}

func (t *Transport) Stop() {
	t.stop()
}

func (t *Transport) cleaner() {
	for {
		select {
		case <-t.globalCtx.Done():
			return
		case <-time.After(3 * time.Second):
		}

		sites := make(map[string]*siteInfo, len(t.activeSites))
		t.mx.RLock()
		for s, info := range t.activeSites {
			sites[s] = info
		}
		t.mx.RUnlock()

		now := time.Now().Unix()
		for s, info := range sites {
			if info.mx.TryLock() {
				// stop bags that was not used for > 5 min
				if atomic.LoadInt64(&info.LastUsed)+300 < now {
					switch act := info.Actor.(type) {
					case *bagInfo:
						t.mx.Lock()
						if t.activeSites[s] == info {
							delete(t.activeSites, s)
						}
						t.mx.Unlock()
						act.downloader.Close()
						act.torrent.Stop()
						log.Println("STOPPED UNUSED BAG", hex.EncodeToString(act.torrent.BagID))
					}
				}
				info.mx.Unlock()
			}
		}
	}
}

func (t *Transport) connectRLDP(ctx context.Context, key ed25519.PublicKey, addr, host string) (RLDP, error) {
	a, err := Connector(ctx, addr, key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init adnl for rldp connection %s, err: %w", addr, err)
	}

	r := newRLDP(a)
	r.SetOnQuery(t.getRLDPQueryHandler(r))
	r.SetOnDisconnect(t.removeRLDP(r, host))

	return r, nil
}

func (t *Transport) removeRLDP(rl RLDP, host string) func() {
	return func() {
		t.mx.RLock()
		r := t.activeSites[host]
		t.mx.RUnlock()

		if r == nil {
			return
		}

		r.mx.Lock()
		defer r.mx.Unlock()

		if act, ok := r.Actor.(*rldpInfo); ok {
			act.destroyClient(rl)
		}
	}
}

func (r *rldpInfo) destroyClient(rl RLDP) {
	rl.Close()

	if r.ActiveClient == rl {
		r.ActiveClient = nil
	}
}

func (t *Transport) getRLDPQueryHandler(r RLDP) func(transferId []byte, query *rldp.Query) error {
	return func(transferId []byte, query *rldp.Query) error {
		switch req := query.Data.(type) {
		case rldphttp.GetNextPayloadPart:
			t.mx.RLock()
			stream := t.activeRequests[hex.EncodeToString(req.ID)]
			t.mx.RUnlock()

			if stream == nil {
				return fmt.Errorf("unknown request id %s", hex.EncodeToString(req.ID))
			}

			part, err := handleGetPart(req, stream)
			if err != nil {
				return fmt.Errorf("handle part err: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err = r.SendAnswer(ctx, query.MaxAnswerSize, query.ID, transferId, part)
			cancel()
			if err != nil {
				return fmt.Errorf("failed to send answer: %w", err)
			}

			if part.IsLast {
				t.mx.Lock()
				delete(t.activeRequests, hex.EncodeToString(req.ID))
				t.mx.Unlock()
				_ = stream.Data.Close()
			}

			return nil
		}
		return fmt.Errorf("unexpected query type %s", reflect.TypeOf(query.Data))
	}
}

func handleGetPart(req rldphttp.GetNextPayloadPart, stream *payloadStream) (*rldphttp.PayloadPart, error) {
	stream.mx.Lock()
	defer stream.mx.Unlock()

	offset := int(req.Seqno * req.MaxChunkSize)
	if offset != stream.nextOffset {
		return nil, fmt.Errorf("failed to get part for stream %s, incorrect offset %d, should be %d", hex.EncodeToString(req.ID), offset, stream.nextOffset)
	}

	var last bool
	data := make([]byte, req.MaxChunkSize)
	n, err := stream.Data.Read(data)
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("failed to read chunk %d, err: %w", req.Seqno, err)
		}
		last = true
	}
	stream.nextOffset += n

	return &rldphttp.PayloadPart{
		Data:    data[:n],
		Trailer: nil, // TODO: trailer
		IsLast:  last,
	}, nil
}

func (s *siteInfo) prepare(t *Transport, request *http.Request) (err error) {
	select {
	case <-t.globalCtx.Done():
		return t.globalCtx.Err()
	default:
	}

	host := request.Host
	if host == "" {
		host = request.URL.Host
	}

	if s.Actor == nil {
		s.Actor, err = t.resolve(request.Context(), host)
		if err != nil {
			return err
		}
	}

	switch act := s.Actor.(type) {
	case *bagInfo:
		atomic.StoreInt64(&s.LastUsed, time.Now().Unix())
	case *rldpInfo:
		if atomic.LoadInt64(&s.LastUsed)+30 < time.Now().Unix() {
			// if last used more than 30 seconds ago,
			// we have a chance of stuck udp socket,
			// so we just reinit connection
			go act.ActiveClient.Close() // close async because of lock

			// set it nil now to reassign
			act.ActiveClient = nil
		}

		if act.ActiveClient == nil {
			act.ActiveClient, err = t.connectRLDP(request.Context(), act.ID, act.Addr, host)
			if err != nil {
				// resolve again
				s.Actor = nil
				return s.prepare(t, request)
			}
			atomic.StoreInt64(&s.LastUsed, time.Now().Unix())
		}
	}
	return nil
}

func (t *Transport) RoundTrip(request *http.Request) (_ *http.Response, err error) {
	host := request.Host
	if host == "" {
		host = request.URL.Host
	}

	t.mx.Lock()
	site := t.activeSites[host]
	if site == nil {
		site = &siteInfo{}
		t.activeSites[host] = site
	}
	t.mx.Unlock()

	var rldpClient RLDP
	var torrent *bagInfo

	tm := time.Now()
	site.mx.Lock()
	err = site.prepare(t, request)
	log.Println("prepare took:", time.Since(tm).String())

	if err != nil {
		site.mx.Unlock()
		return nil, fmt.Errorf("failed to connect to site: %w", err)
	}

	switch act := site.Actor.(type) {
	case *rldpInfo:
		rldpClient = act.ActiveClient
	case *bagInfo:
		torrent = act
	}
	site.mx.Unlock()

	if rldpClient != nil {
		resp, err := t.doRldpHttp(rldpClient, host, request)
		if err != nil {
			return nil, fmt.Errorf("failed to request rldp-http site: %w", err)
		}
		return resp, nil
	}

	resp, err := t.doTorrent(torrent, request, site)
	if err != nil {
		return nil, fmt.Errorf("failed to request file from storage: %w", err)
	}
	return resp, nil
}

func (t *Transport) doTorrent(bag *bagInfo, request *http.Request, si *siteInfo) (*http.Response, error) {
	fileName := request.URL.Path
	if strings.HasPrefix(fileName, "/") {
		fileName = fileName[1:]
	}

	if fileName == "" {
		fileName = "index.html"
	}

	if request.Body != nil {
		tmp := make([]byte, 4096)
		for { // discard body
			_, err := request.Body.Read(tmp)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
		}
	}

	fileInfo, err := bag.torrent.GetFileOffsets(fileName)
	if err != nil {
		return &http.Response{
			Status:        "Not Found",
			StatusCode:    404,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        map[string][]string{},
			ContentLength: 0,
			Trailer:       map[string][]string{},
			Request:       request,
		}, nil
	}

	var typ string
	if strings.Contains(fileName, ".") {
		ext := strings.Split(fileName, ".")
		typ = typeByExtension(ext[len(ext)-1])
	}
	if typ == "" {
		typ = "application/octet-stream"
	}

	fileLastIndex := fileInfo.Size
	if fileLastIndex > 0 {
		fileLastIndex -= 1
	}
	hasRange, from, to, err := t.parseRange(request, fileLastIndex)
	if err != nil {
		log.Println("invalid range:", err.Error())
		return &http.Response{
			Status:        "Invalid range",
			StatusCode:    416,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        map[string][]string{},
			ContentLength: 0,
			Trailer:       map[string][]string{},
			Request:       request,
		}, nil
	}

	pieces := make([]uint32, 0, (fileInfo.ToPiece-fileInfo.FromPiece)+1)
	piecesMap := make(map[uint32]bool, cap(pieces))

	var offFrom, offTo uint64 = 0, 0
	for piece := fileInfo.FromPiece; piece <= fileInfo.ToPiece; piece++ {
		sz := bag.torrent.Info.PieceSize
		if piece == fileInfo.ToPiece {
			sz = fileInfo.ToPieceOffset
		}
		if piece == fileInfo.FromPiece {
			sz -= fileInfo.FromPieceOffset
		}

		offTo += uint64(sz)
		if offTo >= from && offFrom <= to {
			piecesMap[piece] = true
			pieces = append(pieces, piece)
		}
		offFrom = offTo
	}

	httpResp := &http.Response{
		Status:        "OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        map[string][]string{},
		ContentLength: int64((to + 1) - from),
		Trailer:       map[string][]string{},
		Request:       request,
	}

	if hasRange {
		httpResp.StatusCode = http.StatusPartialContent
		httpResp.Status = http.StatusText(http.StatusPartialContent)

		httpResp.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, to, fileInfo.Size))
		httpResp.Header.Set("Content-Length", fmt.Sprint((to+1)-from))
	} else {
		httpResp.Header.Set("Content-Length", fmt.Sprint(fileInfo.Size))
		httpResp.Header.Set("Accept-Ranges", "bytes")
	}
	httpResp.Header.Set("Content-Type", typ)

	if len(pieces) > 0 {
		fetch := storage.NewPreFetcher(request.Context(), bag.torrent, bag.downloader, func(event storage.Event) {}, 0, 8, 50, pieces)
		stream := newDataStreamer()
		httpResp.Body = stream

		go func() {
			defer fetch.Stop()

			err := t.proxyOrdered(request.Context(), fileInfo, piecesMap, fetch, stream, si, bag.torrent.Info.PieceSize, from, to)
			if err != nil {
				_ = stream.Close()
				log.Println("download ordered err: %w", err)
				return
			}
			stream.Finish()
		}()
	}

	return httpResp, nil
}

func (t *Transport) doRldpHttp(client RLDP, host string, request *http.Request) (*http.Response, error) {
	qid := make([]byte, 32)
	_, err := rand.Read(qid)
	if err != nil {
		return nil, err
	}

	req := rldphttp.Request{
		ID:      qid,
		Method:  request.Method,
		URL:     request.URL.String(),
		Version: "HTTP/1.1",
		Headers: []rldphttp.Header{
			{
				Name:  "Host",
				Value: host,
			},
		},
	}

	if request.ContentLength > 0 {
		req.Headers = append(req.Headers, rldphttp.Header{
			Name:  "Content-Length",
			Value: fmt.Sprint(request.ContentLength),
		})
	}

	for k, v := range request.Header {
		for _, hdr := range v {
			req.Headers = append(req.Headers, rldphttp.Header{
				Name:  k,
				Value: hdr,
			})
		}
	}

	if request.Body != nil {
		stream := newDataStreamer()

		// chunked stream reader
		go func() {
			defer request.Body.Close()

			var n int
			for {
				buf := make([]byte, 4096)
				n, err = request.Body.Read(buf)
				if err != nil {
					if errors.Is(err, io.EOF) {
						_, err = stream.Write(buf[:n])
						if err == nil {
							stream.Finish()
							break
						}
					}
					_ = stream.Close()
					break
				}

				_, err = stream.Write(buf[:n])
				if err != nil {
					_ = stream.Close()
					break
				}
			}
		}()

		t.mx.Lock()
		t.activeRequests[hex.EncodeToString(qid)] = &payloadStream{
			Data:      stream,
			ValidTill: time.Now().Add(15 * time.Second),
		}
		t.mx.Unlock()

		defer func() {
			t.mx.Lock()
			delete(t.activeRequests, hex.EncodeToString(qid))
			t.mx.Unlock()
		}()
	}

	var res rldphttp.Response
	err = client.DoQuery(request.Context(), _RLDPMaxAnswerSize, req, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to query http over rldp: %w", err)
	}

	httpResp := &http.Response{
		Status:        res.Reason,
		StatusCode:    int(res.StatusCode),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        map[string][]string{},
		ContentLength: -1,
		Trailer:       map[string][]string{},
		Request:       request,
	}

	for _, header := range res.Headers {
		httpResp.Header[header.Name] = []string{header.Value}
	}

	if ln, ok := request.Header["Content-Length"]; ok && len(ln) > 0 {
		httpResp.ContentLength, err = strconv.ParseInt(ln[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content length: %w", err)
		}
	}

	withPayload := !res.NoPayload && (httpResp.StatusCode < 300 || httpResp.StatusCode >= 400)

	dr := newDataStreamer()
	httpResp.Body = dr

	if withPayload {
		if httpResp.ContentLength > 0 && httpResp.ContentLength < (1<<22) {
			dr.buf = make([]byte, 0, httpResp.ContentLength)
		}

		go func() {
			seqno := int32(0)
			for withPayload {
				var part rldphttp.PayloadPart
				err := client.DoQuery(request.Context(), _RLDPMaxAnswerSize*1000, rldphttp.GetNextPayloadPart{
					ID:           qid,
					Seqno:        seqno,
					MaxChunkSize: _ChunkSize * 100,
				}, &part)
				if err != nil {
					_ = dr.Close()
					return
				}

				for _, tr := range part.Trailer {
					httpResp.Trailer[tr.Name] = []string{tr.Value}
				}

				withPayload = !part.IsLast
				_, err = dr.Write(part.Data)
				if err != nil {
					_ = dr.Close()
					return
				}

				if part.IsLast {
					dr.Finish()
				}

				seqno++
			}
		}()
	} else {
		dr.Finish()
	}

	return httpResp, nil
}

func (t *Transport) resolve(ctx context.Context, host string) (_ any, err error) {
	var id []byte
	var inStorage bool
	if strings.HasSuffix(host, ".adnl") {
		id, err = rldphttp.ParseADNLAddress(host[:len(host)-5])
		if err != nil {
			return nil, fmt.Errorf("failed to parse adnl address %s, err: %w", host, err)
		}
	} else if strings.HasSuffix(host, ".bag") {
		id, err = hex.DecodeString(host[:len(host)-4])
		if err != nil {
			return nil, fmt.Errorf("failed to parse bag id %s, err: %w", host, err)
		}
		inStorage = true
	} else {
		tm := time.Now()
		lookupCtx, stopLookup := context.WithCancel(ctx)
		ch := make(chan *dns.Domain, 3)
		for i := 0; i < 3; i++ { // do parallel lookup on diff nodes to speedup
			go func(i int) {
				for {
					// each new thread has bigger timeout, to cover users with high ping
					resolveCtx, cancel := context.WithTimeout(lookupCtx, time.Duration((i+1)*2)*time.Second)
					domain, err := t.resolver.Resolve(resolveCtx, host)
					cancel()
					if err != nil {
						if lookupCtx.Err() != nil {
							return
						}

						if errors.Is(err, dns.ErrNoSuchRecord) {
							ch <- nil
							return
						}
						log.Println("domain", host, "resolve err: ", err.Error())
						continue
					}

					ch <- domain
					return
				}
			}(i)
		}

		var domain *dns.Domain
		select {
		case domain = <-ch:
			stopLookup()
			if domain == nil {
				return nil, fmt.Errorf("domain %s resolve err: %w", host, dns.ErrNoSuchRecord)
			}
		case <-lookupCtx.Done():
			stopLookup() // to turn off warning
			return nil, fmt.Errorf("failed to resolve domain %s in ton dns", host)
		}
		log.Println("resolve domain", host, "took:", time.Since(tm).String())

		id, inStorage = domain.GetSiteRecord()
	}

	if inStorage {
		log.Println("SEARCHING FOR BAG ID", hex.EncodeToString(id), "OF", host)

		torrent := storage.NewTorrent("", t.store, t.storageConnector)
		torrent.BagID = id

		_ = t.store.SetTorrent(torrent)

		if err = torrent.Start(true, false, false); err != nil {
			return nil, fmt.Errorf("failed to start bag %s, err: %w", host, err)
		}
		log.Println("STARTING FOR BAG ID", hex.EncodeToString(id), "OF", host)

		downloader, err := t.storageConnector.CreateDownloader(t.globalCtx, torrent, 1, 3)
		if err != nil {
			return nil, fmt.Errorf("failed to create downloader for storage bag of %s, err: %w", host, err)
		}

		log.Println("BAG FOUND", hex.EncodeToString(id), "OF", host)
		return &bagInfo{
			torrent:    torrent,
			downloader: downloader,
		}, nil
	}

	log.Println("RESOLVING TON SITE", host, "NODE", hex.EncodeToString(id), "ADDRESS")

	addresses, pubKey, err := t.dht.FindAddresses(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find address of %s (%s) in DHT, err: %w", host, hex.EncodeToString(id), err)
	}

	log.Println("TON SITE", host, "NODE", hex.EncodeToString(id), "ADDRESS RESOLVED")

	var addr string
	var client RLDP
	var triedAddresses []string
	for _, v := range addresses.Addresses {
		addr = fmt.Sprintf("%s:%d", v.IP.String(), v.Port)

		log.Println("CONNECTING TO TON SITE", host, "NODE", hex.EncodeToString(id), "USING ADDRESS", addr)

		// find working rldp node addr
		client, err = t.connectRLDP(ctx, pubKey, addr, host)
		if err != nil {
			log.Println("CONNECTION TO TON SITE", host, "NODE", hex.EncodeToString(id), "USING ADDRESS", addr, "FAILED")

			triedAddresses = append(triedAddresses, addr)
			continue
		}

		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rldp servers %s of host %s, err: %w", triedAddresses, host, err)
	}

	log.Println("TON SITE", host, "NODE", hex.EncodeToString(id), "CONNECTED", addr)

	info := &rldpInfo{
		ActiveClient: client,
		ID:           pubKey,
		Addr:         addr,
	}
	return info, nil
}

func (t *Transport) proxyOrdered(ctx context.Context, file *storage.FileInfo,
	piecesMap map[uint32]bool, fetch *storage.PreFetcher, stream *dataStreamer, si *siteInfo,
	pieceSz uint32, from, to uint64) error {
	var err error
	var currentPieceId uint32
	var currentPiece []byte

	notEmptyFile := file.FromPiece != file.ToPiece || file.FromPieceOffset != file.ToPieceOffset
	if notEmptyFile {
		var toOff uint64
		var wasFirst bool
		for piece := file.FromPiece; piece <= file.ToPiece; piece++ {
			sz := pieceSz
			if piece == file.ToPiece {
				sz = file.ToPieceOffset
			}
			if piece == file.FromPiece {
				sz -= file.FromPieceOffset
			}
			toOff += uint64(sz)

			if !piecesMap[piece] {
				continue
			}

			if piece != currentPieceId || currentPiece == nil {
				if currentPiece != nil {
					fetch.Free(currentPieceId)
				}

				atomic.StoreInt64(&si.LastUsed, time.Now().Unix())

				currentPiece, _, err = fetch.Get(ctx, piece)
				if err != nil {
					return fmt.Errorf("failed to download piece %d: %w", piece, err)
				}

				currentPieceId = piece
			}
			part := currentPiece
			if piece == file.ToPiece {
				part = part[:file.ToPieceOffset]
			}
			if piece == file.FromPiece {
				part = part[file.FromPieceOffset:]
			}

			toOffIdx := toOff - 1
			if toOffIdx > to {
				diff := toOffIdx - to
				part = part[:len(part)-int(diff)]
			}

			fromOff := toOff - uint64(sz)
			if !wasFirst && from > fromOff {
				part = part[from-fromOff:]
			}
			wasFirst = true

			_, err = stream.Write(part)
			if err != nil {
				return fmt.Errorf("failed to write piece %d: %w", piece, err)
			}
		}
	}
	if err != nil {
		return err
	}

	if currentPiece != nil {
		fetch.Free(currentPieceId)
	}
	return nil
}

func (t *Transport) parseRange(request *http.Request, max uint64) (hasRange bool, from uint64, to uint64, err error) {
	rng := request.Header.Get("Range")
	if len(rng) > 6 && strings.HasPrefix(rng, "bytes=") {
		ranges := strings.SplitN(rng[6:], ",", 2)
		if len(ranges) > 1 {
			return false, 0, 0, fmt.Errorf("multiple ranges not supported")
		}

		rngArr := strings.SplitN(ranges[0], "-", 2)
		if len(rngArr) != 2 {
			return false, 0, 0, fmt.Errorf("invalid range format")
		}

		if rngArr[0] != "" {
			from, err = strconv.ParseUint(rngArr[0], 10, 64)
			if err != nil {
				return false, 0, 0, err
			}
			if from > max {
				return false, 0, 0, fmt.Errorf("invalid from range, over max")
			}
		}

		if rngArr[1] != "" {
			to, err = strconv.ParseUint(rngArr[1], 10, 64)
			if err != nil {
				return false, 0, 0, err
			}

			if to > max {
				return false, 0, 0, fmt.Errorf("invalid to range, over max")
			}
		} else {
			to = max
		}

		if from > to {
			return false, 0, 0, fmt.Errorf("invalid range, from > to (%d > %d)", from, to)
		}
		return true, from, to, nil
	}
	return false, 0, max, nil
}
