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
	"github.com/xssnick/tonutils-go/adnl/storage"
	"github.com/xssnick/tonutils-go/tl"
	"github.com/xssnick/tonutils-go/ton/dns"
	"io"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	Query(ctx context.Context, req, result tl.Serializable) error
	SetDisconnectHandler(handler func(addr string, key ed25519.PublicKey))
	SetCustomMessageHandler(handler func(msg *adnl.MessageCustom) error)
	SendCustomMessage(ctx context.Context, req tl.Serializable) error
	Close()
}

type Storage interface {
	CreateDownloader(ctx context.Context, bagId []byte, desiredMinPeersNum, threadsPerPeer int) (_ storage.TorrentDownloader, err error)
}

var Connector = func(ctx context.Context, addr string, peerKey ed25519.PublicKey, ourKey ed25519.PrivateKey) (ADNL, error) {
	return adnl.Connect(ctx, addr, peerKey, ourKey)
}

var newRLDP = func(a ADNL) RLDP {
	return rldp.NewClient(a)
}

type siteInfo struct {
	Actor any

	LastUsed time.Time
	mx       sync.RWMutex
}

type rldpInfo struct {
	ActiveClient RLDP

	ID   ed25519.PublicKey
	Addr string
}

type Transport struct {
	dht      DHT
	resolver Resolver
	storage  Storage

	activeSites map[string]*siteInfo

	activeRequests map[string]*payloadStream
	mx             sync.RWMutex
}

func NewTransport(dht DHT, resolver Resolver, store Storage) *Transport {
	storage.Logger = log.Println
	t := &Transport{
		dht:            dht,
		resolver:       resolver,
		storage:        store,
		activeRequests: map[string]*payloadStream{},
		activeSites:    map[string]*siteInfo{},
	}
	return t
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
	if s.Actor == nil {
		s.Actor, err = t.resolve(request.Context(), request.Host)
		if err != nil {
			return err
		}
		s.LastUsed = time.Now()
	}

	switch act := s.Actor.(type) {
	case *rldpInfo:
		if s.LastUsed.Add(30 * time.Second).Before(time.Now()) {
			// if last used more than 30 seconds ago,
			// we have a chance of stuck udp socket,
			// so we just reinit connection
			go act.ActiveClient.Close() // close async because of lock

			// set it nil now to reassign
			act.ActiveClient = nil
		}

		if act.ActiveClient == nil {
			act.ActiveClient, err = t.connectRLDP(request.Context(), act.ID, act.Addr, request.Host)
			if err != nil {
				// resolve again
				s.Actor = nil
				return s.prepare(t, request)
			}
			s.LastUsed = time.Now()
		}
	}
	return nil
}

func (t *Transport) RoundTrip(request *http.Request) (_ *http.Response, err error) {
	t.mx.Lock()
	site := t.activeSites[request.Host]
	if site == nil {
		site = &siteInfo{}
		t.activeSites[request.Host] = site
	}
	t.mx.Unlock()

	var rldpClient RLDP
	var torrent storage.TorrentDownloader

	site.mx.Lock()
	err = site.prepare(t, request)
	if err != nil {
		site.mx.Unlock()
		return nil, fmt.Errorf("failed to connect to site: %w", err)
	}

	switch act := site.Actor.(type) {
	case *rldpInfo:
		rldpClient = act.ActiveClient
	case storage.TorrentDownloader:
		torrent = act
	}
	site.mx.Unlock()

	if rldpClient != nil {
		resp, err := t.doRldpHttp(rldpClient, request)
		if err != nil {
			return nil, fmt.Errorf("failed to request rldp-http site: %w", err)
		}
		return resp, nil
	}

	resp, err := t.doTorrent(torrent, request)
	if err != nil {
		return nil, fmt.Errorf("failed to request file from storage: %w", err)
	}
	return resp, nil
}

func (t *Transport) doTorrent(dow storage.TorrentDownloader, request *http.Request) (*http.Response, error) {
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

	fileInfo := dow.GetFileOffsets(fileName)
	if fileInfo == nil {
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

	httpResp := &http.Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        map[string][]string{},
		ContentLength: int64(fileInfo.Size),
		Trailer:       map[string][]string{},
		Request:       request,
	}
	httpResp.Header.Set("Content-Length", fmt.Sprint(fileInfo.Size))

	var typ string
	if strings.Contains(fileName, ".") {
		ext := strings.Split(fileName, ".")
		typ = typeByExtension(ext[len(ext)-1])
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	httpResp.Header.Set("Content-Type", typ)

	type partResult struct {
		part uint32
		data []byte
	}

	// scale depends on file size
	var threads = 1
	threads += int(fileInfo.ToPiece-fileInfo.FromPiece) / 50
	if threads > 12 {
		threads = 12
	}
	wantNodes := int(fileInfo.ToPiece-fileInfo.FromPiece) / 600
	if wantNodes > 1 {
		// TODO: not decrease if small file in parallel
		// dow.SetDesiredMinNodesNum(wantNodes)
		// TODO: fix slowdown on bad nodes
	}

	threadsCtx, stopThreads := context.WithCancel(request.Context())
	ch := make(chan uint32, threads*10)
	chResults := make(chan partResult, threads)
	for tr := 0; tr < threads; tr++ {
		go func() {
			for {
				var i uint32
				select {
				case <-threadsCtx.Done():
					return
				case i = <-ch:
				}

				tm := time.Now()
				buf, err := dow.DownloadPiece(threadsCtx, i)
				if err != nil {
					return
				}
				log.Println("DOWNLOADED PIECE ", i, "/", fileInfo.ToPiece, "TOOK:", time.Since(tm).String(), runtime.NumGoroutine())

				if i == fileInfo.ToPiece {
					buf = buf[:fileInfo.ToPieceOffset]
				}
				if i == fileInfo.FromPiece {
					buf = buf[fileInfo.FromPieceOffset:]
				}

				chResults <- partResult{part: i, data: buf}
			}
		}()
	}

	preloadPartsNum := uint32(threads)
	if preloadPartsNum > fileInfo.ToPiece-fileInfo.FromPiece {
		preloadPartsNum = fileInfo.ToPiece - fileInfo.FromPiece
	}
	for i := fileInfo.FromPiece; i <= preloadPartsNum; i++ {
		ch <- i
	}

	stream := newDataStreamer()
	httpResp.Body = stream

	go func() {
		cache := map[uint32][]byte{}
		for i := fileInfo.FromPiece; i <= fileInfo.ToPiece; {
			part, ok := cache[i]
			if !ok {
				block := <-chResults
				if block.part != i {
					cache[block.part] = block.data
					// not ++ i pointer, since it is not needed part
					continue
				}
				part = block.data
			} else {
				delete(cache, i)
			}

			_, err := stream.Write(part)
			if err != nil {
				_ = stream.Close()
				return
			}

			// add task to load one more part, since we have free processing slot
			if preloadPartsNum <= fileInfo.ToPiece {
				ch <- preloadPartsNum
				preloadPartsNum++
			}
			i++
		}
		stream.Finish()
		stopThreads()
	}()

	return httpResp, nil
}

func (t *Transport) doRldpHttp(client RLDP, request *http.Request) (*http.Response, error) {
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
				Value: request.Host,
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
		var domain *dns.Domain
		for i := 0; i < 3; i++ {
			domain, err = t.resolver.Resolve(ctx, host)
			if err != nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host %s, err: %w", host, err)
		}

		id, inStorage = domain.GetSiteRecord()
	}

	if inStorage {
		log.Println("SEARCHING FOR BAG ID", hex.EncodeToString(id))

		dow, err := t.storage.CreateDownloader(ctx, id, 1, 3)
		if err != nil {
			return nil, fmt.Errorf("failed to create downloader for storage bag of %s, err: %w", host, err)
		}
		log.Println("BAG FOUND", hex.EncodeToString(id))

		return dow, nil
	}

	addresses, pubKey, err := t.dht.FindAddresses(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find address of %s (%s) in DHT, err: %w", host, hex.EncodeToString(id), err)
	}

	var addr string
	var client RLDP
	var triedAddresses []string
	for _, v := range addresses.Addresses {
		addr = fmt.Sprintf("%s:%d", v.IP.String(), v.Port)

		// find working rldp node addr
		client, err = t.connectRLDP(ctx, pubKey, addr, host)
		if err != nil {
			triedAddresses = append(triedAddresses, addr)
			continue
		}

		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rldp servers %s of host %s, err: %w", triedAddresses, host, err)
	}

	info := &rldpInfo{
		ActiveClient: client,
		ID:           pubKey,
		Addr:         addr,
	}
	return info, nil
}
