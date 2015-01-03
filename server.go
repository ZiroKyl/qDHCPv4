package main

import(
	//"strings"
	//"bytes"
	"log"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

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
	stage    IPStage;
	ip       net.IP;
	mac	     MAC48;
	updated  bool;	//need on tLeaseEnd
	pervL   *linkedLease;
	nextL   *linkedLease;
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

	iCurrTLeaseEnd int;

	_CONN         *ServeConn;
	_SERVER_IP     net.IP;
	_START_IP      net.IP;
	_OPTIONS       dhcp.Options;
	_T_LEASE_END []int16;

	channelReq <-chan reqChan;/*don't close()*/

	leases            []linkedLease;
	clients map[MAC48] *linkedLease;
}
func(H *DhcpHandler) Init(conn *ServeConn, serverIP, startIP net.IP, rangeIP int, options dhcp.Options, tLeaseEnd []int16, channelReq/*don't close()*/ <-chan reqChan){
	H._CONN        = conn;
	H._SERVER_IP   = serverIP;
	H._START_IP    = startIP;
	H._OPTIONS     = options;
	H._T_LEASE_END = tLeaseEnd;

	H.iCurrTLeaseEnd = 0;

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
	var dEmptyReserve    = time.Duration(len(H.leases))*5*time.Millisecond;
	var dNotEmptyReserve =                              5*time.Millisecond;

	var timerClearReserve = time.After(dEmptyReserve);

	var tNow = time.Now().Local();
	var tNowM = tNow.Hour() * 60 + tNow.Minute();

	H.correctICurrTLeaseEnd(tNowM);

	var durClearIssued = time.Duration(modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()])-tNowM))*time.Minute;
	var tClearIssuedStage = -40*time.Second;
	var timerClearIssuedStage = time.After(durClearIssued + tClearIssuedStage);
	for{
		select{
		// clear old Issued leases on schedule
		case <- timerClearIssuedStage:
			switch(tClearIssuedStage){
			//set update=false on Issued leases
			case -40*time.Second:
				for i := range H.leases{
					if H.leases[i].stage == IP_Issued {
						H.leases[i].updated = false;
					}
				}

				durClearIssued = 40*time.Second;
				tClearIssuedStage = 2*time.Minute + 20*time.Second;
				timerClearIssuedStage = time.After(durClearIssued + tClearIssuedStage);
			//delete Issued leases if update=false
			case 2*time.Minute + 20*time.Second:
				for i := range H.leases{
					if H.leases[i].stage == IP_Issued && H.leases[i].updated == false {
						H.DeleteClientL(&H.leases[i]);
					}
				}

				tNow = time.Now().Local();
				tNowM = tNow.Hour() * 60 + tNow.Minute();

				H.correctICurrTLeaseEnd(tNowM);

				durClearIssued = time.Duration(modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()])-tNowM))*time.Minute;
				tClearIssuedStage = -40*time.Second;
				timerClearIssuedStage = time.After(durClearIssued + tClearIssuedStage);
			}

		//clear old Discover-Offer leases on timeout
		case <- timerClearReserve:
			if oldLease:=H.qLastReserveL; oldLease != nil { // time to Request lease = 0..durClearReserve
				H.UniPutLease(H.UniRemoveLease(oldLease), IP_Free);
				H.DeleteMAC(oldLease.mac);

				timerClearReserve = time.After(dNotEmptyReserve);
			}else{
				timerClearReserve = time.After(dEmptyReserve);
			}

		case req := <- H.channelReq:
			serveOut(*H._CONN, req.req, H.Handler(req.req, req.reqType, req.options), req.addr);
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
func(H *DhcpHandler) DeleteClientL(lease/*!nil*/ *linkedLease) net.IP{
	//TODO: on debug "if H.clients[mac].stage == IP_Free { panic("Bug#-5464985-"); }"
	H.UniPutLease(H.UniRemoveLease(lease), IP_Free);
	H.DeleteMAC(lease.mac);

	return nil;
}


func(H *DhcpHandler) correctICurrTLeaseEnd(tNowM int){
	for _ = range H._T_LEASE_END {
		// L a R -> len(a-L)+len(R-a)=len(R-L)
		if modOneDay(tNowM-int(H._T_LEASE_END[H.iCurrTLeaseEnd])) + modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()])-tNowM) ==
		   modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()]-H._T_LEASE_END[H.iCurrTLeaseEnd])) &&
		   modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()])-tNowM)!=0 {
			break;
		}
		H.iCurrTLeaseEnd = H.NextTLeaseEnd();
	}
}
func(H *DhcpHandler) NextTLeaseEnd() int{
	return (H.iCurrTLeaseEnd+1)%len(H._T_LEASE_END);
}
func modOneDay(minutes int) int{
	return (24*60+minutes)%(24*60);
}
func(H *DhcpHandler) LeaseDuration(ip net.IP) time.Duration{
	var tNow = time.Now().Local();
	var tNowM = tNow.Hour()*60 + tNow.Minute();

	H.correctICurrTLeaseEnd(tNowM);

	//minimize load: lease_time += ip_offset
	return time.Duration(modOneDay(int(H._T_LEASE_END[H.NextTLeaseEnd()])-tNowM))*time.Minute + time.Duration((dhcp.IPRange(H._START_IP, ip)*60)/len(H.leases))*time.Second;
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
					return dhcp.ReplyPacket(req, dhcp.Offer, H._SERVER_IP, outIP, H.LeaseDuration(outIP),
											H._OPTIONS.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]));
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
					if outIP != nil { H.clients[reqMAC].updated = true; //need on tLeaseEnd
						              return dhcp.ReplyPacket(req, dhcp.ACK, H._SERVER_IP, outIP, H.LeaseDuration(outIP),
						                                      H._OPTIONS.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]));
					}else           { return dhcp.ReplyPacket(req, dhcp.NAK, H._SERVER_IP, nil,   0, nil) }
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
					return dhcp.ReplyPacket(req, dhcp.ACK, H._SERVER_IP, nil, 0, H._OPTIONS.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]));
				}else{
					log.Println("Interesting (Inform msg)...");
				}

				return nil;
			},
		})[msgType]();
};

