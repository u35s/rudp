package rudp

import (
	"bytes"
	"errors"
	"log"
)

const (
	TYPE_IGNORE = iota
	TYPE_EOF
	TYPE_CORRUPT
	TYPE_REQUEST
	TYPE_MISSING
	TYPE_NORMAL
)

const (
	//GENERAL_PACKAGE = 512
	GENERAL_PACKAGE = 128
	MAX_PACKAGE     = 0x7fff - TYPE_NORMAL
)

type TypeCorrupt int

func (this TypeCorrupt) Error() error {
	switch this {
	case TYPE_CORRUPT_EOF:
		return errors.New("EOF")
	case TYPE_CORRUPT_REMOTE_EOF:
		return errors.New("REMOTE EOF")
	case TYPE_CORRUPT_CORRUPT:
		return errors.New("CORRUPT")
	case TYPE_CORRUPT_MSG_LARGE:
		return errors.New("SEND MSG TOOL LARGE")
	case TYPE_CORRUPT_MSG_SIZE:
		return errors.New("RECIVE MSG SIZE ERROR")
	default:
		return nil
	}
}

const (
	TYPE_CORRUPT_NIL TypeCorrupt = iota
	TYPE_CORRUPT_EOF
	TYPE_CORRUPT_REMOTE_EOF
	TYPE_CORRUPT_CORRUPT
	TYPE_CORRUPT_MSG_LARGE
	TYPE_CORRUPT_MSG_SIZE
)

type Package struct {
	Next *Package
	Bts  []byte
}

type packageBuffer struct {
	tmp  bytes.Buffer
	head *Package
	tail *Package
}

func (tmp *packageBuffer) packRequest(id, tag int) {
	if tmp.tmp.Len()+3 > GENERAL_PACKAGE {
		tmp.newPackage()
	}
	tmp.fillHeader(tag, id)
}

func (tmp *packageBuffer) fillHeader(head, id int) {
	if head < 128 {
		tmp.tmp.WriteByte(byte(head))
	} else {
		tmp.tmp.WriteByte(byte(((head & 0x7f00) >> 8) | 0x80))
		tmp.tmp.WriteByte(byte(head & 0xff))
	}
	tmp.tmp.WriteByte(byte((id & 0xff00) >> 8))
	tmp.tmp.WriteByte(byte(id & 0xff))
}

func (tmp *packageBuffer) packMessage(m *message) {
	if m.buf.Len()+4 > GENERAL_PACKAGE {
		if tmp.tmp.Len() > 0 {
			tmp.newPackage()
			tmp.tmp.Reset()
		}
		tmp.fillHeader(m.buf.Len()+TYPE_NORMAL, m.id)
		tmp.tmp.Write(m.buf.Bytes())
		tmp.newPackage()
	}
	if m.buf.Len()+4+tmp.tmp.Len() > GENERAL_PACKAGE {
		tmp.newPackage()
	}
	tmp.fillHeader(m.buf.Len()+TYPE_NORMAL, m.id)
	tmp.tmp.Write(m.buf.Bytes())
}

func (tmp *packageBuffer) newPackage() {
	p := &Package{Bts: make([]byte, tmp.tmp.Len())}
	copy(p.Bts, tmp.tmp.Bytes())
	tmp.tmp.Reset()
	if tmp.tail == nil {
		tmp.head = p
		tmp.tail = p
	} else {
		tmp.tail.Next = p
		tmp.tail = p
	}
}

var corruptTick int = 5
var expiredTick int = 1024
var sendDelayTick int = 1

func SetTick(corrupt, expired, sendDelay int) {
	corruptTick = corrupt
	expiredTick = expired
	sendDelayTick = sendDelay
}

func New() *Rudp {
	return &Rudp{corruptTick: corruptTick, expiredTick: expiredTick, sendDelayTick: sendDelayTick,
		addSendAgain: make(chan int, 1<<10),
	}
}

type Rudp struct {
	recvQueue messageQueue
	recvIDMin int
	recvIDMax int

	sendQueue    messageQueue
	sendHistory  messageQueue
	addSendAgain chan int
	sendID       int

	corrupt TypeCorrupt

	corruptTick   int
	expiredTick   int
	sendDelayTick int

	currentTick       int
	lastRecvTick      int
	lastExpiredTick   int
	lastSendDelayTick int
}

func (this *Rudp) Recv(bts []byte) (int, error) {
	if err := this.corrupt; err != TYPE_CORRUPT_NIL {
		return 0, err.Error()
	}
	m := this.recvQueue.pop(this.recvIDMin)
	if m == nil {
		return 0, nil
	}
	this.recvIDMin++
	copy(bts, m.buf.Bytes())
	return m.buf.Len(), nil
}

func (this *Rudp) Send(bts []byte) (n int, err error) {
	if err := this.corrupt; err != TYPE_CORRUPT_NIL {
		return 0, err.Error()
	}
	if len(bts) > MAX_PACKAGE {
		return 0, TYPE_CORRUPT_MSG_LARGE.Error()
	}
	m := &message{}
	m.buf.Write(bts)
	m.id = this.sendID
	this.sendID++
	m.tick = this.currentTick
	this.sendQueue.push(m)
	return len(bts), nil
}

func (this *Rudp) Update(tick int) *Package {
	if this.corrupt != TYPE_CORRUPT_NIL {
		return nil
	}
	this.currentTick += tick
	if this.currentTick >= this.lastExpiredTick+this.expiredTick {
		this.lastExpiredTick = this.currentTick
		this.clearSendExpired()
	}
	if this.currentTick >= this.lastSendDelayTick+this.sendDelayTick {
		this.lastSendDelayTick = this.currentTick
		return this.outPut()
	}
	if this.currentTick >= this.lastRecvTick+this.corruptTick {
		this.corrupt = TYPE_CORRUPT_CORRUPT
	}
	return nil
}

type message struct {
	next *message
	buf  bytes.Buffer
	id   int
	tick int
}

type messageQueue struct {
	head *message
	tail *message
}

func (this *messageQueue) pop(id int) *message {
	if this.head == nil {
		return nil
	}
	m := this.head
	if id >= 0 && m.id != id {
		return nil
	}
	this.head = m.next
	m.next = nil
	if this.head == nil {
		this.tail = nil
	}
	return m
}

func (this *messageQueue) push(m *message) {
	if this.tail == nil {
		this.head = m
		this.tail = m
	} else {
		this.tail.next = m
		this.tail = m
	}
}

func (this *Rudp) getID(max int, bt1, bt2 byte) int {
	n1, n2 := int(bt1), int(bt2)
	id := n1*256 + n2
	id |= max & ^0xffff
	if id < max-0x8000 {
		log.Printf("id < max-0x8000 ,id %v max %v,cur %v",
			id, max, id+0x10000)
		id += 0x10000
	} else if id > max+0x8000 {
		log.Printf("id > max+0x8000 ,id %v max %v,cur %v",
			id, max, id-0x10000)
		id -= 0x10000
	}
	return id
}

func (this *Rudp) outPut() *Package {
	var tmp packageBuffer
	tmp.tmp.Grow(GENERAL_PACKAGE)
	this.reqMissing(&tmp)
	this.replyRequest(&tmp)
	this.sendMessage(&tmp)
	if tmp.head == nil && tmp.tmp.Len() == 0 {
		tmp.tmp.WriteByte(byte(TYPE_IGNORE))
	}
	tmp.newPackage()
	return tmp.head
}

func (this *Rudp) Input(bts []byte) {
	sz := len(bts)
	if sz > 0 {
		this.lastRecvTick = this.currentTick
	}
	for sz > 0 {
		len := int(bts[0])
		if len > 127 {
			if sz <= 1 {
				this.corrupt = TYPE_CORRUPT_MSG_SIZE
				return
			}
			len = (len*256 + int(bts[1])) & 0x7fff
			bts = bts[2:]
			sz -= 2
		} else {
			bts = bts[1:]
			sz -= 1
		}
		switch len {
		case TYPE_IGNORE:
		case TYPE_EOF:
			this.corrupt = TYPE_CORRUPT_EOF
		case TYPE_CORRUPT:
			this.corrupt = TYPE_CORRUPT_REMOTE_EOF
			return
		case TYPE_REQUEST, TYPE_MISSING:
			if sz < 2 {
				this.corrupt = TYPE_CORRUPT_MSG_SIZE
				return
			}
			exe := this.addRequest
			max := this.sendID
			if len == TYPE_MISSING {
				exe = this.addMissing
				max = this.recvIDMax
			}
			exe(this.getID(max, bts[0], bts[1]))
			bts = bts[2:]
			sz -= 2
		default:
			len -= TYPE_NORMAL
			if sz < len+2 {
				this.corrupt = TYPE_CORRUPT_MSG_SIZE
				return
			}
			this.insertMessage(this.getID(this.recvIDMax, bts[0], bts[1]), bts[2:len+2])
			bts = bts[len+2:]
			sz -= len + 2
		}
	}

}

func (this *Rudp) insertMessage(id int, bts []byte) {
	if id < this.recvIDMin {
		return
	}
	if id > this.recvIDMax || this.recvQueue.head == nil {
		m := &message{}
		m.buf.Write(bts)
		m.id = id
		this.recvQueue.push(m)
		this.recvIDMax = id
	} else {
		m := this.recvQueue.head
		last := &this.recvQueue.head
		for m != nil {
			if m.id > id {
				tmp := &message{}
				tmp.buf.Write(bts)
				tmp.id = id
				tmp.next = m
				*last = tmp
				return
			}
			last = &m.next
			m = m.next
		}
	}
}

func (this *Rudp) sendMessage(tmp *packageBuffer) {
	m := this.sendQueue.head
	for m != nil {
		tmp.packMessage(m)
		m = m.next
	}
	if this.sendQueue.head != nil {
		if this.sendHistory.tail == nil {
			this.sendHistory = this.sendQueue
		} else {
			this.sendHistory.tail.next = this.sendQueue.head
			this.sendHistory.tail = this.sendQueue.tail
		}
		this.sendQueue.head = nil
		this.sendQueue.tail = nil
	}
}
func (this *Rudp) clearSendExpired() {
	m := this.sendHistory.head
	for m != nil {
		if m.tick >= this.lastExpiredTick {
			break
		}
		m = m.next
	}
	this.sendHistory.head = m
	if m == nil {
		this.sendHistory.tail = nil
	}
}

func (this *Rudp) addRequest(id int) {
	log.Printf("add request %v", id)
	this.addSendAgain <- id
}

func (this *Rudp) addMissing(id int) {
	log.Printf("add missing %v", id)
	this.insertMessage(id, []byte{})
}

func (this *Rudp) replyRequest(tmp *packageBuffer) {
	for {
		select {
		case id := <-this.addSendAgain:
			history := this.sendHistory.head
			if id < this.recvIDMin {
				//already recv,ignore
				log.Printf("already recv %v", id)
				continue
			}
			for {
				if history == nil || id < history.id {
					//expired
					log.Printf("send again miss %v", id)
					tmp.packRequest(id, TYPE_MISSING)
					break
				} else if id == history.id {
					log.Printf("send again %v", id)
					tmp.packMessage(history)
					break
				}
				history = history.next
			}
		default:
			return
		}
	}
}

func (this *Rudp) reqMissing(tmp *packageBuffer) {
	id := this.recvIDMin
	m := this.recvQueue.head
	for m != nil {
		if m.id < id {
			break
		}
		if m.id > id {
			for i := id; i < m.id; i++ {
				log.Printf("req miss %v", i)
				tmp.packRequest(i, TYPE_REQUEST)
			}
		}
		id = m.id + 1
		m = m.next
	}
}
