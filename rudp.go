package rudp

import (
	"bytes"
	"errors"
	"time"
)

const (
	TYPE_PING = iota
	TYPE_EOF
	TYPE_CORRUPT
	TYPE_REQUEST
	TYPE_MISSING
	TYPE_NORMAL
)

const (
	MAX_MSG_HEAD    = 4
	GENERAL_PACKAGE = 576 - 60 - 8
	MAX_PACKAGE     = 0x7fff - TYPE_NORMAL
)

type Error int

const (
	ERROR_NIL Error = iota
	ERROR_EOF
	ERROR_REMOTE_EOF
	ERROR_CORRUPT
	ERROR_MSG_SIZE
)

func (this Error) Error() error {
	switch this {
	case ERROR_EOF:
		return errors.New("EOF")
	case ERROR_REMOTE_EOF:
		return errors.New("remote EOF")
	case ERROR_CORRUPT:
		return errors.New("corrupt")
	case ERROR_MSG_SIZE:
		return errors.New("recive msg size error")
	default:
		return nil
	}
}

type Package struct {
	Next *Package
	Bts  []byte
}

type packageBuffer struct {
	tmp  bytes.Buffer
	num  int
	head *Package
	tail *Package
}

func (tmp *packageBuffer) packRequest(min, max int, tag int) {
	if tmp.tmp.Len()+5 > GENERAL_PACKAGE {
		tmp.newPackage()
	}
	tmp.tmp.WriteByte(byte(tag))
	tmp.tmp.WriteByte(byte((min & 0xff00) >> 8))
	tmp.tmp.WriteByte(byte(min & 0xff))
	tmp.tmp.WriteByte(byte((max & 0xff00) >> 8))
	tmp.tmp.WriteByte(byte(max & 0xff))
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
	if m.buf.Len()+4+tmp.tmp.Len() >= GENERAL_PACKAGE {
		tmp.newPackage()
	}
	tmp.fillHeader(m.buf.Len()+TYPE_NORMAL, m.id)
	tmp.tmp.Write(m.buf.Bytes())
}
func (tmp *packageBuffer) newPackage() {
	if tmp.tmp.Len() <= 0 {
		return
	}
	p := &Package{Bts: make([]byte, tmp.tmp.Len())}
	copy(p.Bts, tmp.tmp.Bytes())
	tmp.tmp.Reset()
	tmp.num++
	if tmp.tail == nil {
		tmp.head = p
		tmp.tail = p
	} else {
		tmp.tail.Next = p
		tmp.tail = p
	}
}

func New() *Rudp {
	return &Rudp{reqSendAgain: make(chan [2]int, 1<<10), addSendAgain: make(chan [2]int, 1<<10), recvSkip: make(map[int]int)}
}

type Rudp struct {
	recvQueue    messageQueue
	recvSkip     map[int]int
	reqSendAgain chan [2]int
	recvIDMin    int
	recvIDMax    int

	sendQueue    messageQueue
	sendHistory  messageQueue
	addSendAgain chan [2]int
	sendID       int

	corrupt Error

	currentTick       int
	lastRecvTick      int
	lastExpiredTick   int
	lastSendDelayTick int
}

func (this *Rudp) Recv(bts []byte) (int, error) {
	if err := this.corrupt; err != ERROR_NIL {
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
	if err := this.corrupt; err != ERROR_NIL {
		return 0, err.Error()
	}
	if len(bts) > MAX_PACKAGE {
		return 0, nil
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
	if this.corrupt != ERROR_NIL {
		return nil
	}
	this.currentTick += tick
	if this.currentTick >= this.lastExpiredTick+expiredTick {
		this.lastExpiredTick = this.currentTick
		this.clearSendExpired()
	}
	if this.currentTick >= this.lastRecvTick+corruptTick {
		this.corrupt = ERROR_CORRUPT
	}
	if this.currentTick >= this.lastSendDelayTick+sendDelayTick {
		this.lastSendDelayTick = this.currentTick
		return this.outPut()
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
	num  int
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
	this.num--
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
	this.num++
}

func (this *Rudp) getID(max int, bt1, bt2 byte) int {
	n1, n2 := int(bt1), int(bt2)
	id := n1*256 + n2
	id |= max & ^0xffff
	if id < max-0x8000 {
		id += 0x10000
		dbg("id < max-0x8000 ,net %v,id %v,min %v,max %v,cur %v",
			n1*256+n2, id, this.recvIDMin, max, id+0x10000)
	} else if id > max+0x8000 {
		id -= 0x10000
		dbg("id > max-0x8000 ,net %v,id %v,min %v,max %v,cur %v",
			n1*256+n2, id, this.recvIDMin, max, id+0x10000)
	}
	return id
}

func (this *Rudp) outPut() *Package {
	var tmp packageBuffer
	this.reqMissing(&tmp)
	this.replyRequest(&tmp)
	this.sendMessage(&tmp)
	if tmp.head == nil && tmp.tmp.Len() == 0 {
		tmp.tmp.WriteByte(byte(TYPE_PING))
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
				this.corrupt = ERROR_MSG_SIZE
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
		case TYPE_PING:
			this.checkMissing(false)
		case TYPE_EOF:
			this.corrupt = ERROR_EOF
		case TYPE_CORRUPT:
			this.corrupt = ERROR_REMOTE_EOF
			return
		case TYPE_REQUEST, TYPE_MISSING:
			if sz < 4 {
				this.corrupt = ERROR_MSG_SIZE
				return
			}
			exe := this.addRequest
			max := this.sendID
			if len == TYPE_MISSING {
				exe = this.addMissing
				max = this.recvIDMax
			}
			exe(this.getID(max, bts[0], bts[1]), this.getID(max, bts[2], bts[3]))
			bts = bts[4:]
			sz -= 4
		default:
			len -= TYPE_NORMAL
			if sz < len+2 {
				this.corrupt = ERROR_MSG_SIZE
				return
			}
			this.insertMessage(this.getID(this.recvIDMax, bts[0], bts[1]), bts[2:len+2])
			bts = bts[len+2:]
			sz -= len + 2
		}
	}
	this.checkMissing(false)
}

func (this *Rudp) checkMissing(direct bool) {
	head := this.recvQueue.head
	if head != nil && head.id > this.recvIDMin {
		nano := int(time.Now().UnixNano())
		last := this.recvSkip[this.recvIDMin]
		if !direct && last == 0 {
			this.recvSkip[this.recvIDMin] = nano
			dbg("miss start %v-%v,max %v", this.recvIDMin, head.id-1, this.recvIDMax)
		} else if direct || last+missingTime < nano {
			delete(this.recvSkip, this.recvIDMin)
			this.reqSendAgain <- [2]int{this.recvIDMin, head.id - 1}
			dbg("req miss %v-%v,direct %v,wait num %v",
				this.recvIDMin, head.id-1, direct, this.recvQueue.num)
		}
	}
}

func (this *Rudp) insertMessage(id int, bts []byte) {
	if id < this.recvIDMin {
		dbg("already recv %v,len %v", id, len(bts))
		return
	}
	delete(this.recvSkip, id)
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
			if m.id == id {
				dbg("repeat recv id %v,len %v", id, len(bts))
			} else if m.id > id {
				tmp := &message{}
				tmp.buf.Write(bts)
				tmp.id = id
				tmp.next = m
				*last = tmp
				this.recvQueue.num++
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

func (this *Rudp) addRequest(min, max int) {
	dbg("add request %v-%v,max send id %v", min, max, this.sendID)
	this.addSendAgain <- [2]int{min, max}
}

func (this *Rudp) addMissing(min, max int) {
	if max < this.recvIDMin {
		dbg("add missing %v-%v fail,already recv,min %v", min, max, this.recvIDMin)
		return
	}
	if min > this.recvIDMin {
		dbg("add missing %v-%v fail, more than min %v", min, max, this.recvIDMin)
		return
	}
	head := 0
	if this.recvQueue.head != nil {
		head = this.recvQueue.head.id
	}
	dbg("add missing %v-%v,min %v,head %v", min, max, this.recvIDMin, head)
	this.recvIDMin = max + 1
	this.checkMissing(true)
}

func (this *Rudp) replyRequest(tmp *packageBuffer) {
	for {
		select {
		case again := <-this.addSendAgain:
			history := this.sendHistory.head
			min, max := again[0], again[1]
			if history == nil || max < history.id {
				dbg("send again miss %v-%v,send max %v", min, max, this.sendID)
				tmp.packRequest(min, max, TYPE_MISSING)
			} else {
				var start, end, num int
				for {
					if history == nil || max < history.id {
						//expired
						break
					} else if min <= history.id {
						tmp.packMessage(history)
						if start == 0 {
							start = history.id
						}
						end = history.id
						num++
					}
					history = history.next
				}
				if min < start {
					tmp.packRequest(min, start-1, TYPE_MISSING)
					dbg("send again miss %v-%v,send max %v", min, start-1, this.sendID)
				}
				dbg("send again %v-%v of %v-%v,all %v,max send id %v", start, end, min, max, num, this.sendID)
			}
		default:
			return
		}
	}
}

func (this *Rudp) reqMissing(tmp *packageBuffer) {
	for {
		select {
		case req := <-this.reqSendAgain:
			tmp.packRequest(req[0], req[1], TYPE_REQUEST)
		default:
			return
		}
	}
}
