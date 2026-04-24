package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	opReply = 1
	opQuery = 2004
	opMsg   = 2013
)

var reqCounter int32

func main() {
	listenAddr := "127.0.0.1:27018"
	targetAddr := "127.0.0.1:27017"

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Proxy: %s -> %s", listenAddr, targetAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, targetAddr)
	}
}

type connState struct {
	mu       sync.Mutex
	reqIDMap map[int32]int32 // proxyReqID -> clientReqID
}

func handleConn(client net.Conn, targetAddr string) {
	defer client.Close()

	server, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer server.Close()

	state := &connState{reqIDMap: make(map[int32]int32)}
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		proxyC2S(client, server, state)
	}()
	go func() {
		defer wg.Done()
		proxyS2C(server, client, state)
	}()

	wg.Wait()
}

func readMsg(r io.Reader) (header [16]byte, body []byte, err error) {
	_, err = io.ReadFull(r, header[:])
	if err != nil {
		return
	}
	msgLen := int(binary.LittleEndian.Uint32(header[:4]))
	if msgLen < 16 || msgLen > 48*1024*1024 {
		err = fmt.Errorf("bad len: %d", msgLen)
		return
	}
	body = make([]byte, msgLen-16)
	_, err = io.ReadFull(r, body)
	return
}

func nextID() int32 { return atomic.AddInt32(&reqCounter, 1) }

func proxyC2S(client, server io.ReadWriter, state *connState) {
	for {
		header, body, err := readMsg(client)
		if err != nil {
			return
		}

		opcode := binary.LittleEndian.Uint32(header[12:16])
		clientReqID := int32(binary.LittleEndian.Uint32(header[4:8]))

		if opcode == opQuery {
			cmd, collection, queryBson := parseQuery(body)
			log.Printf("-> OP_QUERY %s (cmd=%s, reqID=%d)", collection, cmd, clientReqID)

			// Intercept getnonce
			if cmd == "getnonce" {
				reply := makeReply(clientReqID, bsonDoc(
					bsonStr("nonce", "deadbeef01234567"),
					bsonF64("ok", 1),
				))
				client.Write(reply)
				continue
			}

			// Extract db name
			db := collection
			if i := strings.Index(collection, "."); i >= 0 {
				db = collection[:i]
			}

			// Translate to OP_MSG
			proxyReqID := nextID()
			state.mu.Lock()
			state.reqIDMap[proxyReqID] = clientReqID
			state.mu.Unlock()

			modified := injectDB(queryBson, db)
			msg := makeOpMsg(proxyReqID, modified)
			server.Write(msg)
			continue
		}

		// Forward other opcodes as-is
		fwd := make([]byte, 16+len(body))
		copy(fwd, header[:])
		copy(fwd[16:], body)
		server.Write(fwd)
	}
}

func proxyS2C(server, client io.ReadWriter, state *connState) {
	for {
		header, body, err := readMsg(server)
		if err != nil {
			return
		}

		opcode := binary.LittleEndian.Uint32(header[12:16])
		responseTo := int32(binary.LittleEndian.Uint32(header[8:12]))

		if opcode == opMsg {
			// Look up original client request ID
			state.mu.Lock()
			clientReqID, ok := state.reqIDMap[responseTo]
			if ok {
				delete(state.reqIDMap, responseTo)
			}
			state.mu.Unlock()

			if !ok {
				clientReqID = responseTo
			}

			doc := extractMsgDoc(body)
			if doc != nil {
				log.Printf("<- OP_MSG->REPLY (responseTo=%d)", clientReqID)
				reply := makeReply(clientReqID, doc)
				client.Write(reply)
				continue
			}
		}

		// Forward as-is
		fwd := make([]byte, 16+len(body))
		copy(fwd, header[:])
		copy(fwd[16:], body)
		client.Write(fwd)
	}
}

func parseQuery(body []byte) (cmd, collection string, queryBson []byte) {
	if len(body) < 12 {
		return
	}
	nullIdx := bytes.IndexByte(body[4:], 0)
	if nullIdx < 0 {
		return
	}
	collection = string(body[4 : 4+nullIdx])
	off := 4 + nullIdx + 1 + 8
	if off+4 > len(body) {
		return
	}
	bLen := int(binary.LittleEndian.Uint32(body[off : off+4]))
	if off+bLen > len(body) {
		return
	}
	queryBson = body[off : off+bLen]
	cmd = firstKey(queryBson)
	return
}

func firstKey(b []byte) string {
	if len(b) < 6 {
		return ""
	}
	i := bytes.IndexByte(b[5:], 0)
	if i < 0 {
		return ""
	}
	return string(b[5 : 5+i])
}

func extractMsgDoc(body []byte) []byte {
	if len(body) < 6 {
		return nil
	}
	// flagBits(4) + kind(1) + doc
	if body[4] != 0 {
		return nil
	}
	off := 5
	if off+4 > len(body) {
		return nil
	}
	dLen := int(binary.LittleEndian.Uint32(body[off : off+4]))
	if off+dLen > len(body) {
		return nil
	}
	return body[off : off+dLen]
}

func makeReply(responseTo int32, doc []byte) []byte {
	// header(16) + responseFlags(4) + cursorID(8) + startingFrom(4) + numberReturned(4) + doc
	replyBody := make([]byte, 20+len(doc))
	binary.LittleEndian.PutUint32(replyBody[0:4], 8) // awaitCapable
	binary.LittleEndian.PutUint32(replyBody[16:20], 1)
	copy(replyBody[20:], doc)

	total := 16 + len(replyBody)
	msg := make([]byte, total)
	binary.LittleEndian.PutUint32(msg[0:4], uint32(total))
	binary.LittleEndian.PutUint32(msg[4:8], uint32(nextID()))
	binary.LittleEndian.PutUint32(msg[8:12], uint32(responseTo))
	binary.LittleEndian.PutUint32(msg[12:16], opReply)
	copy(msg[16:], replyBody)
	return msg
}

func makeOpMsg(reqID int32, doc []byte) []byte {
	body := make([]byte, 5+len(doc))
	body[4] = 0
	copy(body[5:], doc)

	total := 16 + len(body)
	msg := make([]byte, total)
	binary.LittleEndian.PutUint32(msg[0:4], uint32(total))
	binary.LittleEndian.PutUint32(msg[4:8], uint32(reqID))
	binary.LittleEndian.PutUint32(msg[12:16], opMsg)
	copy(msg[16:], body)
	return msg
}

func injectDB(orig []byte, db string) []byte {
	if len(orig) < 5 {
		return orig
	}
	v := []byte(db)
	elem := make([]byte, 0, 9+len(v))
	elem = append(elem, 0x02)
	elem = append(elem, '$', 'd', 'b', 0)
	l := make([]byte, 4)
	binary.LittleEndian.PutUint32(l, uint32(len(v)+1))
	elem = append(elem, l...)
	elem = append(elem, v...)
	elem = append(elem, 0)

	result := make([]byte, 0, len(orig)+len(elem))
	result = append(result, orig[:len(orig)-1]...)
	result = append(result, elem...)
	result = append(result, 0)
	binary.LittleEndian.PutUint32(result[:4], uint32(len(result)))
	return result
}

// BSON helpers
func bsonDoc(elems ...[]byte) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 4))
	for _, e := range elems {
		buf.Write(e)
	}
	buf.WriteByte(0)
	r := buf.Bytes()
	binary.LittleEndian.PutUint32(r[:4], uint32(len(r)))
	return r
}

func bsonStr(key, val string) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x02)
	buf.WriteString(key)
	buf.WriteByte(0)
	l := make([]byte, 4)
	binary.LittleEndian.PutUint32(l, uint32(len(val)+1))
	buf.Write(l)
	buf.WriteString(val)
	buf.WriteByte(0)
	return buf.Bytes()
}

func bsonF64(key string, val float64) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x01)
	buf.WriteString(key)
	buf.WriteByte(0)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(val))
	buf.Write(b)
	return buf.Bytes()
}
