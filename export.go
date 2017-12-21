package rudp

const (
	//GENERAL_PACKAGE = 512
	GENERAL_PACKAGE = 128
	MAX_PACKAGE     = 0x7fff - 4
)

type Package struct {
	Next *Package
	Bts  []byte
}

func New(sendDelay, expiredTime int) *Rudp {
	return &Rudp{sendDelay: sendDelay, expiredTime: expiredTime}
}

type Rudp struct {
	freeList messageQueue

	recvQueue messageQueue
	recvIDMin int
	recvIDMax int

	sendQueue   messageQueue
	sendHistory messageQueue
	sendAgain   array
	sendID      int

	corrupt         int
	currentTick     int
	lastSendTick    int
	lastExpiredTick int

	sendDelay   int
	expiredTime int
}

func (this *Rudp) Recv(bts []byte) int {
	if this.corrupt > 0 {
		this.corrupt = 0
		return -1
	}
	m := this.recvQueue.pop(this.recvIDMin)
	if m == nil {
		return 0
	}
	this.recvIDMin++
	sz, tmp := m.buf.Len(), m.buf.Bytes()
	if sz > 0 {
		copy(bts, tmp)
	}
	m.reset()
	this.freeList.push(m)
	return sz
}

func (this *Rudp) Send(bts []byte) {
	assert(len(bts) <= MAX_PACKAGE)
	m := this.newMessage(bts)
	m.id = this.sendID
	this.sendID++
	m.tick = this.currentTick
	this.sendQueue.push(m)
}

func (this *Rudp) Update(bts []byte, tick int) *Package {
	this.currentTick += tick
	this.extractPackage(bts)
	if this.currentTick >= this.lastExpiredTick+this.expiredTime {
		this.clearSendExpired()
		this.lastExpiredTick = this.currentTick
	}
	if this.currentTick >= this.lastSendTick+this.sendDelay {
		this.lastSendTick = this.currentTick
		return this.genOutpackage()
	}
	return nil
}
