package main

import(
	"strconv"
	//"strings"
	//"bytes"
	"log"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

/*
type lease struct {
	nic    string    // Client's CHAddr
	expiry time.Time // When the lease expires
}

type DHCPHandler struct {
	ip            net.IP        // Server IP to use
	options       dhcp.Options  // Options to send to DHCP Clients
	start         net.IP        // Start of IP range to distribute
	leaseRange    int           // Number of IPs to distribute (starting from start)
	leaseDuration time.Duration // Lease period
	leases        map[int]lease // Map to keep track of leases
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {

	case dhcp.Discover:
		free, nic := -1, p.CHAddr().String()
		for i, v := range h.leases { // Find previous lease
			if v.nic == nic {
				free = i
				goto reply
			}
		}
		if free = h.freeLease(); free == -1 {
			return
		}
	reply:
		return dhcp.ReplyPacket(p, dhcp.Offer, h.ip, dhcp.IPAdd(h.start, free), h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	case dhcp.Request:
		if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.ip) {
			return nil // Message not for this dhcp server
		}
		if reqIP := net.IP(options[dhcp.OptionRequestedIPAddress]); len(reqIP) == 4 {
			if leaseNum := dhcp.IPRange(h.start, reqIP) - 1; leaseNum >= 0 && leaseNum < h.leaseRange {
				if l, exists := h.leases[leaseNum]; !exists || l.nic == p.CHAddr().String() {
					h.leases[leaseNum] = lease{nic: p.CHAddr().String(), expiry: time.Now().Add(h.leaseDuration)}
					return dhcp.ReplyPacket(p, dhcp.ACK, h.ip, net.IP(options[dhcp.OptionRequestedIPAddress]), h.leaseDuration,
						h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
				}
			}
		}
		return dhcp.ReplyPacket(p, dhcp.NAK, h.ip, nil, 0, nil)

	case dhcp.Release, dhcp.Decline:
		nic := p.CHAddr().String()
	for i, v := range h.leases {
		if v.nic == nic {
			delete(h.leases, i)
			break
		}
	}
	}
	return nil
}

func (h *DHCPHandler) freeLease() int {
	now := time.Now()
	b := rand.Intn(h.leaseRange) // Try random first
	for _, v := range [][]int{[]int{b, h.leaseRange}, []int{0, b}} {
		for i := v[0]; i < v[1]; i++ {
			if l, ok := h.leases[i]; !ok || l.expiry.Before(now) {
				return i
			}
		}
	}
	return -1
}

// Example using DHCP with a single network interface device
func ExampleHandler() {
	serverIP := net.IP{172, 30, 0, 1}
	handler := &DHCPHandler{
		ip:            serverIP,
		leaseDuration: 2 * time.Hour,
		start:         net.IP{172, 30, 0, 2},
		leaseRange:    50,
		leases:        make(map[int]lease, 10),
		options: dhcp.Options{
			dhcp.OptionSubnetMask:       []byte{255, 255, 240, 0},
			dhcp.OptionRouter:           []byte(serverIP), // Presuming Server is also your router
			dhcp.OptionDomainNameServer: []byte(serverIP), // Presuming Server is also your DNS server
		},
	}
	//log.Fatal(dhcp.ListenAndServe(handler))
	// log.Fatal(dhcp.ListenAndServeIf("eth0",handler)) // Select interface on multi interface device
}
*/


type MAC48 [6]byte;

var zeroMAC = MAC48{0,0,0,0,0,0};

type IPStage byte;
const (
	IP_Removed IPStage = iota;
	IP_Free;
	IP_Reserved;
	IP_Issued;
)

type linkedLease struct{
	stage   IPStage;
	ip      net.IP;
	mac	    MAC48;
	pervL  *linkedLease;
	nextL  *linkedLease;
};

func(L *linkedLease) IP() net.IP{
	if L==nil { return nil;
	}else     { return L.ip; }
}

// ServeConn is the bare minimum connection functions required by Serve()
// It allows you to create custom connections for greater control,
// such as ServeIfConn (see serverif.go), which locks to a given interface.
type ServeConn interface {
	ReadFrom(b []byte) (n int, addr net.Addr, err error)
	WriteTo(b []byte, addr net.Addr) (n int, err error)
}

type reqChan struct{
	req dhcp.Packet;
	reqType dhcp.MessageType;
	options dhcp.Options;
	addr net.Addr;
}


type DhcpHandler struct{
	qLastFreeL         *linkedLease; //    Queue-->
	qFirstFreeL        *linkedLease; // -->Queue
	qFirstReserveL     *linkedLease;
	qLastReserveL      *linkedLease;

	_CONN      *ServeConn;
	_SERVER_IP  net.IP;
	_START_IP   net.IP;
	_OPTIONS    dhcp.Options;

	channelReq <-chan reqChan;/*don't close()*/

	leases            []linkedLease;
	clients map[MAC48] *linkedLease;
}
func(H *DhcpHandler) Init(conn *ServeConn, serverIP, startIP net.IP, rangeIP int, options dhcp.Options, channelReq/*don't close()*/ <-chan reqChan){
	H._CONN      = conn;
	H._SERVER_IP = serverIP;
	H._START_IP  = startIP;
	H._OPTIONS   = options;

	H.channelReq = channelReq;

	H.leases  = make(          []linkedLease, rangeIP);
	H.clients = make(map[MAC48] *linkedLease, rangeIP);

	{//Init leases
		H.leases[len(H.leases)-1].pervL =  nil;
		H.leases[len(H.leases)-1].nextL = &H.leases[len(H.leases)-2];
		H.leases[len(H.leases)-1].stage =  IP_Free;
		H.leases[len(H.leases)-1].ip    =  dhcp.IPAdd(H._START_IP, len(H.leases)-1);

		for i:=len(H.leases)-2; i>0; i-- {
			H.leases[i].pervL = &H.leases[i+1];
			H.leases[i].nextL = &H.leases[i-1];
			H.leases[i].stage =  IP_Free;
			H.leases[i].ip    =  dhcp.IPAdd(H._START_IP, i);
		}

		H.leases[0].pervL = &H.leases[1];
		H.leases[0].nextL =  nil;
		H.leases[0].stage =  IP_Free;
		H.leases[0].ip    =  H._START_IP;

		H.qLastFreeL    = &H.leases[0];
		H.qFirstFreeL   = &H.leases[len(H.leases)-1];
		H.qFirstReserveL = nil;
		H.qLastReserveL  = nil;
	}//Init leases end
}
func(H *DhcpHandler) GoHandle(){
	var dEmptyReserve    = len(H.leases)*5*time.Millisecond;
	var dNotEmptyReserve =               5*time.Millisecond;
	
	var durClearReserve = dEmptyReserve;
	for{
		select{
		case req := <- H.channelReq:
			H.serveOut(req.req, H.Handler(req.req, req.reqType, req.options), req.addr);

		//clear old Discover-Offer leases on timeout
		case <- time.After(durClearReserve):
			if H.qLastReserveL != nil { // time to Request lease = 0..durClearReserve
				H.UniPutLease(H.UniRemoveLease(H.qLastReserveL), IP_Free);
				H.DeleteMAC(H.qLastReserveL.mac);

				durClearReserve = dNotEmptyReserve;
			}else{
				durClearReserve = dEmptyReserve;
			}
		}
	}
}

func(H *DhcpHandler) Lease(offset int) *linkedLease{
	return &H.leases[offset];
}
func(  *DhcpHandler) removeLease(lease/*!nil*/ *linkedLease, firstLease, lastLease **linkedLease/**(nil),!=*/){
	lease.stage = IP_Removed;

	if lease.nextL != nil { lease.nextL.pervL = lease.pervL; }
	if lease.pervL != nil { lease.pervL.nextL = lease.nextL; }

	if lease == *firstLease { *firstLease  = lease.nextL; }
	if lease == *lastLease  { *lastLease   = lease.pervL; }

	lease.pervL = nil;
	lease.nextL = nil;
}
func(  *DhcpHandler) putLease(lease/*!nil*/ *linkedLease, firstLease, lastLease **linkedLease/**(!nil),==*/, stage IPStage){
	lease.pervL = nil;
	lease.nextL = *firstLease;

	if *firstLease != nil { (*firstLease).pervL = lease;
	}else{ *lastLease = lease; }

	*firstLease = lease;
	lease.stage = stage;
}
func(H *DhcpHandler) GetFreeL(reqL/*!nil*/ *linkedLease) (freeL *linkedLease){
	H.removeLease(reqL, &H.qFirstFreeL, &H.qLastFreeL);
	freeL = reqL;
	H.putLease(freeL, &H.qFirstReserveL, &H.qLastReserveL, IP_Reserved);

	return;
}
func(H *DhcpHandler) LastFreeL() (freeL *linkedLease){
	if H.qLastFreeL == nil { return nil; }

	freeL = H.GetFreeL(H.qLastFreeL);
	return;
}
func(H *DhcpHandler) GetReserveL(resL/*!nil*/ *linkedLease) (issuedL *linkedLease){
	var nilL *linkedLease;

	H.removeLease(resL, &H.qFirstReserveL, &H.qLastReserveL);
	issuedL = resL;
	issuedL.stage = IP_Issued;

	return;
}
//func(H *DhcpHandler) RetIssuedL(issuedL/*!nil*/ *linkedLease){
//	H.putLease(issuedL, &H.qFirstFreeL, &H.qLastFreeL, IP_Free);
//}
func(H *DhcpHandler) UniRemoveLease(lease/*!nil*/ *linkedLease)  *linkedLease{
	switch lease.stage {
	case IP_Free:     H.removeLease(lease, &H.qFirstFreeL,    &H.qLastFreeL);
	case IP_Reserved: H.removeLease(lease, &H.qFirstReserveL, &H.qLastReserveL);
	}
	return lease;
};
func(H *DhcpHandler) UniPutLease(lease/*!nil*/ *linkedLease, stage IPStage) *linkedLease{
	switch stage {
	case IP_Reserved: H.putLease(lease, &H.qFirstReserveL, &H.qLastReserveL, IP_Reserved);
	case IP_Free:     H.putLease(lease, &H.qFirstFreeL,    &H.qLastFreeL,    IP_Free);
	}
	return lease;
}


func(H *DhcpHandler) SetMAC(lease/*!nil*/ *linkedLease, mac MAC48) *linkedLease{
	H.clients[mac] = lease;
	lease.mac = mac;

	return lease;
}
func(H *DhcpHandler) DeleteMAC(mac MAC48){
	H.clients[mac].mac = zeroMAC;
	delete(H.clients, mac);
}

func(H *DhcpHandler) NewClient(lease *linkedLease, mac MAC48) *linkedLease{
	if lease != nil { H.SetMAC(lease, mac); }
	return lease;
}
func(H *DhcpHandler) OldClient(lease/*!nil*/ *linkedLease) *linkedLease{
	//TODO: on debug "if lease.stage == IP_Free { panic("Bug#5464985"); }"
	if lease.stage == IP_Issued { H.UniPutLease(lease, IP_Reserved); }
	return lease;
}
func(H *DhcpHandler) ReSetClient(lease/*!nil*/ *linkedLease, mac MAC48, stage IPStage) *linkedLease{
	H.UniPutLease(H.UniRemoveLease(H.clients[mac]), IP_Free);
	H.SetMAC(H.UniPutLease(H.UniRemoveLease(lease), stage), mac);

	return lease;
}
func(H *DhcpHandler) DeleteClient(mac MAC48) net.IP{
	//TODO: on debug "if H.clients[mac].stage == IP_Free { panic("Bug#-5464985-"); }"
	H.UniPutLease(H.UniRemoveLease(H.clients[mac]), IP_Free);
	H.DeleteMAC(mac);

	return nil;
}


func(H *DhcpHandler) Handler(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {
	return (map[dhcp.MessageType] func() dhcp.Packet{
			dhcp.Discover: func() dhcp.Packet{
				var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
				var reqIP = net.IP(options[dhcp.OptionRequestedIPAddress]);

				if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(net.IPv4zero) { reqIP = tr; }

				if reqIP!=nil { if r:=dhcp.IPRange(H._START_IP, reqIP)-1; !(r>=0 && r<len(H.leases)) { reqIP=nil; } }

				var outIP net.IP;

				var reqMACLease = H.clients[reqMAC];

				switch {
				case reqIP == nil && reqMACLease == nil: outIP = H.NewClient(H.LastFreeL(), reqMAC).IP();
				case reqIP == nil && reqMACLease != nil: outIP = H.OldClient(reqMACLease).IP();
				case reqIP != nil && reqMACLease == nil:
					var reqIPLease = H.Lease(dhcp.IPRange(H._START_IP, reqIP)-1);
					switch reqIPLease.stage {
					case IP_Free:                outIP = H.NewClient(H.GetFreeL(reqIPLease), reqMAC).IP(); //use user IP
					case IP_Reserved, IP_Issued: outIP = H.NewClient(H.LastFreeL(),          reqMAC).IP(); //try send another lease
					}
				case reqIP != nil && reqMACLease != nil:
					var reqIPLease = H.Lease(dhcp.IPRange(H._START_IP, reqIP)-1);
					if reqIPLease.mac == reqMAC {    outIP = H.OldClient(reqMACLease).IP();
					}else{
						switch reqIPLease.stage {
						case IP_Free:                outIP = H.ReSetClient(reqIPLease, reqMAC, IP_Reserved).IP(); //move to user IP
						case IP_Reserved, IP_Issued: outIP = H.OldClient(reqMACLease).IP();
						}
					}
				}

				if outIP != nil {
					return dhcp.ReplyPacket(req, dhcp.Offer, H._SERVER_IP, outIP, time.Hour,
											H._OPTIONS.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]));//TODO: сделать также и в остальных местах
				}

				return nil;
			},
			dhcp.Request: func() dhcp.Packet{
				var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
				var reqIP       = net.IP(options[dhcp.OptionRequestedIPAddress]);
				var reqServerIP = net.IP(options[dhcp.OptionServerIdentifier]);

				if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(net.IPv4zero) { reqIP = tr; }

				//very buggy client
				if reqServerIP == nil && reqIP == nil { return nil; }

				//try correct
				if reqServerIP == nil { reqServerIP = H._SERVER_IP; }
				if reqIP == nil {
					if reqIP = H.clients[reqMAC].IP(); reqIP == nil { return nil; }	//correct fail
				}

				var outIP net.IP;

				var offsetIP = dhcp.IPRange(H._START_IP, reqIP)-1;
				var reqIPInRange = offsetIP>=0 && offsetIP<len(H.leases);
				var reqMACInClients = H.clients[reqMAC] != nil;

				switch {
				case reqIPInRange && reqMACInClients:
					var reqIPLease = H.Lease(offsetIP);
					if reqIPLease.mac == reqMAC {											//TODO: test speed "H.Lease(offsetReqIP)==H.clients[reqMAC]" && "H.clients[reqMAC].lease.Equal(reqL)"
						if reqIPLease.stage == IP_Reserved { H.GetReserveL(reqIPLease); }	//TODO: on debug "if reqIPLease.stage == IP_Free { panic("Bug#+5464985+"); }"
						outIP = reqIP;
					}else{
						switch reqIPLease.stage {
						case IP_Free:                outIP = H.ReSetClient(reqIPLease, reqMAC, IP_Issued).IP(); //move to user IP
						case IP_Reserved, IP_Issued: outIP = H.DeleteClient(reqMAC);
						}
					}
				case reqIPInRange && !reqMACInClients:
					var reqIPLease = H.Lease(offsetIP);
					switch reqIPLease.stage {
					case IP_Free:                outIP = H.SetMAC(H.UniPutLease(H.UniRemoveLease(reqIPLease), IP_Issued), reqMAC).IP(); //use user IP
					case IP_Reserved, IP_Issued: outIP = nil; //user broken off
					}
				case !reqIPInRange &&  reqMACInClients: outIP = H.DeleteClient(reqMAC);
				case !reqIPInRange && !reqMACInClients: outIP = nil;
				}

				if reqServerIP.Equal(H._SERVER_IP) {
					if outIP != nil { return dhcp.ReplyPacket(req, dhcp.ACK, H._SERVER_IP, outIP, time.Hour, nil);
					}else           { return dhcp.ReplyPacket(req, dhcp.NAK, H._SERVER_IP, nil,   time.Hour, nil) }
				}

				return nil;
			},
			dhcp.Decline: func() dhcp.Packet{
				var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());

				if H.clients[reqMAC] != nil {
					log.Println("leases conflict (Decline msg) - leases: " + H.clients[reqMAC].IP().String() + " MAC: " + net.HardwareAddr(reqMAC[:]).String());
					H.DeleteClient(reqMAC);
				}

				return nil;
			},
			dhcp.Release: func() dhcp.Packet{
				var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
				var reqIP  = req.CIAddr();

				if reqIP.Equal(net.IPv4zero) { reqIP = nil; }

				if H.clients[reqMAC] != nil && !(reqIP != nil && !H.clients[reqMAC].IP().Equal(reqIP)) {
					H.DeleteClient(reqMAC);
				}

				return nil;
			},
			dhcp.Inform: func() dhcp.Packet{
				var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
				var reqIP  = req.CIAddr();

				if H.clients[reqMAC].IP().Equal(reqIP) { // + "clients[reqMAC] != nil" :-)
					return dhcp.ReplyPacket(req, dhcp.ACK, H._SERVER_IP, nil, 0, nil);
				}else{
					log.Println("Interesting (Inform msg)...");
				}

				return nil;
			},
		})[msgType]();
};

func(H *DhcpHandler) serveOut(req dhcp.Packet, res dhcp.Packet, addr net.Addr) error {
	if res != nil {
		// If IP not available, broadcast
		ipStr, portStr, err := net.SplitHostPort(addr.String());
		if err != nil {
			return err;
		}

		if net.ParseIP(ipStr).Equal(net.IPv4zero) || req.Broadcast() {
			port, _ := strconv.Atoi(portStr);
			addr = &net.UDPAddr{IP: net.IPv4bcast, Port: port};
		}
		if _, e := (*H._CONN).WriteTo(res, addr); e != nil {
			return e;
		}
	}
	return nil;
}
