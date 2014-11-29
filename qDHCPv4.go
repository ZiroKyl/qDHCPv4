package main

import(
	"fmt"
	//"strings"
	//"bytes"
	"log"
	"net"
	//"math/rand"
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

// http://support.microsoft.com/kb/169289/ru
// https://ru.wikipedia.org/wiki/DHCP#.D0.9F.D1.80.D0.B8.D0.BC.D0.B5.D1.80_.D0.BF.D1.80.D0.BE.D1.86.D0.B5.D1.81.D1.81.D0.B0_.D0.BF.D0.BE.D0.BB.D1.83.D1.87.D0.B5.D0.BD.D0.B8.D1.8F_.D0.B0.D0.B4.D1.80.D0.B5.D1.81.D0.B0

// Host Name:
//android-a70378b9bf61c919
//android-99d3a92f5cb0d61e
//iPhone-Elena
//iPad-Andrey
//Windows-Phone

// Tests:
// http://blog.thecybershadow.net/2013/01/10/dhcp-test-client/
// http://www.opennet.ru/prog/info/3604.shtml
// http://www.ingmarverheij.com/microsoft-vendor-specific-dhcp-options-explained-and-demystified/



func main() {
	fmt.Printf("Hello world!");


	dhcp.ListenAndServe(func() dhcp.Handler {
		//all leases - array + linked queue for last free
		//MAC - map

		type MAC48 [6]byte;

		const (
			IP_Range = 1024;
		)
		var (
			START_IP  = net.IP{194,188,32,1};
			SERVER_IP = net.IP{194,188,64,28};
		)

		var zeroIP  = net.IP{0,0,0,0};
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

		var leases[IP_Range]  linkedLease;
		var qLastFreeL       *linkedLease; //    Queue-->
		var qFirstFreeL      *linkedLease; // -->Queue
		var firstReserveL    *linkedLease;

		{//Init leases
			leases[len(leases)-1].pervL =  nil;
			leases[len(leases)-1].nextL = &leases[len(leases)-2];
			leases[len(leases)-1].stage =  IP_Free;
			leases[len(leases)-1].ip    =  dhcp.IPAdd(START_IP, len(leases)-1);

			for i:=len(leases)-2; i>0; i-- {
				leases[i].pervL = &leases[i+1];
				leases[i].nextL = &leases[i-1];
				leases[i].stage =  IP_Free;
				leases[i].ip    =  dhcp.IPAdd(START_IP, i);
			}

			leases[0].pervL = &leases[1];
			leases[0].nextL =  nil;
			leases[0].stage =  IP_Free;
			leases[0].ip    =  START_IP;

			qLastFreeL    = &leases[0];
			qFirstFreeL   = &leases[len(leases)-1];
			firstReserveL =  nil;
		}//Init leases end

		var Lease = func(offset int) *linkedLease{
			return &leases[offset];
		};
		var removeLease = func(lease/*!nil*/ *linkedLease, firstLease, lastLease **linkedLease/**(nil),!=*/){
			lease.stage = IP_Removed;

			if lease.nextL != nil { lease.nextL.pervL = lease.pervL; }
			if lease.pervL != nil { lease.pervL.nextL = lease.nextL; }

			if lease == *firstLease { *firstLease  = lease.nextL; }
			if lease == *lastLease  { *lastLease   = lease.pervL; }

			lease.pervL = nil;
			lease.nextL = nil;
		};
		var putLease = func(lease/*!nil*/ *linkedLease, firstLease, lastLease **linkedLease/**(!nil),==*/, stage IPStage){
			lease.pervL = nil;
			lease.nextL = *firstLease;

			if *firstLease != nil { (*firstLease).pervL = lease;
			}else{ *lastLease = lease; }

			*firstLease = lease;
			lease.stage = stage;
		};
		var GetFreeL = func(reqL/*!nil*/ *linkedLease) (freeL *linkedLease){
			removeLease(reqL, &qFirstFreeL, &qLastFreeL);
			freeL = reqL;
			putLease(freeL, &firstReserveL, &firstReserveL, IP_Reserved);

			return;
		};
		var LastFreeL = func() (freeL *linkedLease){
			if qLastFreeL == nil { return nil; }

			freeL = GetFreeL(qLastFreeL);
			return;
		};
		var GetReserveL = func(resL/*!nil*/ *linkedLease) (issuedL *linkedLease){
			var nilL *linkedLease;

			removeLease(resL, &firstReserveL, &nilL);
			issuedL = resL;
			issuedL.stage = IP_Issued;

			return;
		};
		//var RetIssuedL = func(issuedL/*!nil*/ *linkedLease){
		//	putLease(issuedL, &qFirstFreeL, &qLastFreeL, IP_Free);
		//};
		var UniRemoveLease = func(lease/*!nil*/ *linkedLease)  *linkedLease{
			var nilL *linkedLease;

			switch lease.stage {
			case IP_Free:     removeLease(lease, &qFirstFreeL,   &qLastFreeL);
			case IP_Reserved: removeLease(lease, &firstReserveL, &nilL      );
			}
			return lease;
		};
		var UniPutLease = func(lease/*!nil*/ *linkedLease, stage IPStage) *linkedLease{
			switch stage {
			case IP_Reserved: putLease(lease, &firstReserveL, &firstReserveL, IP_Reserved);
			case IP_Free:     putLease(lease, &qFirstFreeL,   &qLastFreeL,    IP_Free);
			}
			return lease;
		}

		var clients = map[MAC48] *linkedLease {};

		var SetMAC = func(lease/*!nil*/ *linkedLease, mac MAC48) *linkedLease{
			clients[mac] = lease;
			lease.mac = mac;

			return lease;
		};
		var DeleteMAC = func(mac MAC48){
			clients[mac].mac = zeroMAC;
			delete(clients, mac);
		};

		var NewClient = func(lease *linkedLease, mac MAC48) *linkedLease{
			if lease != nil { SetMAC(lease, mac); }
			return lease;
		};
		var OldClient = func(lease *linkedLease) *linkedLease{
			//TODO: on debug "if lease.stage == IP_Free { panic("Bug#5464985"); }"
			if lease.stage == IP_Issued { UniPutLease(lease, IP_Reserved); }
			return lease;
		};
		var ReSetClient = func(lease/*!nil*/ *linkedLease, mac MAC48, stage IPStage) *linkedLease{
			UniPutLease(UniRemoveLease(clients[mac]), IP_Free);
			SetMAC(UniPutLease(UniRemoveLease(lease), stage), mac);

			return lease;
		};
		var DeleteClient = func(mac MAC48) net.IP{
			//TODO: on debug "if clients[mac].stage == IP_Free { panic("Bug#-5464985-"); }"
			UniPutLease(UniRemoveLease(clients[mac]), IP_Free);
			DeleteMAC(mac);

			return nil;
		};


		//TODO: clear old Discover-Offer leases on timeout
		//TODO: minimize load: lease_time += ip_offset
		//TODO: network IO is async?

		//TODO: разъеденить функции <-------------------------------------VVVVVVV-----(внизу)

		return dhcp.Handler{
			ServeDHCP: func(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {

				/*
				switch android, iPhone, PC, ...
				*/
				//req.
				/*var leasesMobileIP = map[string] string{
					"iPhone":"Apple",
					"":"noname",
				};
				var leasesMobileMAC = map[string] string{
					"iPhone":"Apple",
					"":"noname",
				};
				var leasesPC;
				log.Println(leases[string(bytes.SplitN(options[dhcp.OptionHostName],[]byte{'-'},2)[0])]);*/

				return (map[dhcp.MessageType] func() dhcp.Packet{
					dhcp.Discover: func() dhcp.Packet{
						var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
						var reqIP = net.IP(options[dhcp.OptionRequestedIPAddress]);

						if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(zeroIP) { reqIP = tr; }

						if reqIP!=nil { if r:=dhcp.IPRange(START_IP, reqIP)-1; !(r>=0 && r<IP_Range) { reqIP=nil; } }

						var outIP net.IP;

						var reqMACLease = clients[reqMAC];

						switch {
						case reqIP == nil && reqMACLease == nil: outIP = NewClient(LastFreeL(), reqMAC).ip;
						case reqIP == nil && reqMACLease != nil: outIP = OldClient(reqMACLease).ip;
						case reqIP != nil && reqMACLease == nil:
							var reqIPLease = Lease(dhcp.IPRange(START_IP, reqIP)-1);
							switch reqIPLease.stage {
							case IP_Free:                outIP = NewClient(GetFreeL(reqIPLease), reqMAC).ip; //use user IP
							case IP_Reserved, IP_Issued: outIP = NewClient(LastFreeL(),          reqMAC).ip; //try send another lease
							}
						case reqIP != nil && reqMACLease != nil:
							var reqIPLease = Lease(dhcp.IPRange(START_IP, reqIP)-1);
							if reqIPLease.mac == reqMAC {    outIP = OldClient(reqMACLease).ip;
							}else{
								switch reqIPLease.stage {
								case IP_Free:                outIP = ReSetClient(reqIPLease, reqMAC, IP_Reserved).ip; //move to user IP
								case IP_Reserved, IP_Issued: outIP = OldClient(reqMACLease).ip;
								}
							}
						}

						if outIP != nil { return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, outIP, time.Hour, nil); }

						return nil;
						/*return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clients[reqMAC],
							h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]), nil);*/
					},
					dhcp.Request: func() dhcp.Packet{
						var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
						var reqIP       = net.IP(options[dhcp.OptionRequestedIPAddress]);
						var reqServerIP = net.IP(options[dhcp.OptionServerIdentifier]);

						if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(zeroIP) { reqIP = tr; }

						//very buggy client
						if reqServerIP == nil && reqIP == nil { return nil; }

						//try correct
						if reqServerIP == nil { reqServerIP = SERVER_IP; }
						if reqIP == nil {
							if reqIP = clients[reqMAC].ip; reqIP == nil { return nil; }	//correct fail
						}

						var outIP net.IP;

						var offsetIP = dhcp.IPRange(START_IP, reqIP)-1;
						var reqIPInRange = offsetIP>=0 && offsetIP<IP_Range;
						var reqMACInClients = clients[reqMAC] != nil;

						switch {
						case reqIPInRange && reqMACInClients:
							var reqIPLease = Lease(offsetIP);
							if reqIPLease.mac == reqMAC {										//TODO: test speed "Lease(offsetReqIP)==clients[reqMAC]" && "clients[reqMAC].lease.Equal(reqL)"
								if reqIPLease.stage == IP_Reserved { GetReserveL(reqIPLease); }	//TODO: on debug "if reqIPLease.stage == IP_Free { panic("Bug#+5464985+"); }"
								outIP = reqIP;
							}else{
								switch reqIPLease.stage {
								case IP_Free:                outIP = ReSetClient(reqIPLease, reqMAC, IP_Issued).ip; //move to user IP
								case IP_Reserved, IP_Issued: outIP = DeleteClient(reqMAC);
								}
							}
						case reqIPInRange && !reqMACInClients:
							var reqIPLease = Lease(offsetIP);
							switch reqIPLease.stage {
							case IP_Free:                outIP = SetMAC(UniPutLease(UniRemoveLease(reqIPLease), IP_Issued), reqMAC).ip; //use user IP
							case IP_Reserved, IP_Issued: outIP = nil; //user broken off
							}
						case !reqIPInRange &&  reqMACInClients: outIP = DeleteClient(reqMAC);
						case !reqIPInRange && !reqMACInClients: outIP = nil;
						}

						if reqServerIP.Equal(SERVER_IP) {
							if outIP != nil { return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, outIP, time.Hour, nil);
							}else           { return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil,   time.Hour, nil) }
						}

						return nil;
					},
					dhcp.Decline: func() dhcp.Packet{
						var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());

						if clients[reqMAC] != nil {
							log.Println("leases conflict (Decline msg) - leases: " + clients[reqMAC].ip.String() + " MAC: " + net.HardwareAddr(reqMAC[:]).String());
							DeleteClient(reqMAC);
						}

						return nil;
					},
					dhcp.Release: func() dhcp.Packet{
						var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
						var reqIP  = req.CIAddr();

						if reqIP.Equal(zeroIP) { reqIP = nil; }

						if clients[reqMAC] != nil && !(reqIP != nil && !clients[reqMAC].ip.Equal(reqIP)) {
							DeleteClient(reqMAC);
						}

						return nil;
					},
					dhcp.Inform: func() dhcp.Packet{
						var reqMAC MAC48; copy(reqMAC[:], req.CHAddr());
						var reqIP  = req.CIAddr();

						if clients[reqMAC].ip.Equal(reqIP) { // + "clients[reqMAC] != nil" :-)
							return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, nil, 0, nil);
						}else{
							log.Println("Interesting (Inform msg)...");
						}

						return nil;
					},
				})[msgType]();
			},
		};
	}());
}