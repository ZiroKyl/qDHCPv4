package main

import(
	"fmt"
	//"strings"
	"bytes"
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
		//all IP - array + linked queue for last free
		//MAC - map

		const (
			IP_Range  = 1024;
			START_IP  = net.IP{194,188,32,1};
			SERVER_IP = net.IP{194,188,64,28};
		)

		type Stage byte;
		const (
			IP_Free     Stage = iota;
			IP_Reserved Stage;
			IP_NotFree  Stage;
		)

		type queueIP struct{
			ip            net.IP;
			 mac	net.HardwareAddr;
			 //expiry	time.Time;
			stage         Stage;	//Only for light freeIPTest()
			nextFreeIP   *queueIP;
			beforeFreeIP *queueIP;
		};

		var IP[IP_Range]  queueIP;
		var lastFreeIP   *queueIP;
		var endFreeIP    *queueIP;
		var endReserveIP *queueIP;

		{//Init IP
			IP[len(IP)-1].nextFreeIP = nil;
			IP[len(IP)-1].beforeFreeIP = &IP[len(IP)-2];
			IP[len(IP)-1].stage = IP_Free;
			IP[len(IP)-1].ip = dhcp.IPAdd(START_IP, len(IP)-1);

			for i := len(IP) - 2; i > 0; i-- {
				IP[i].nextFreeIP = &IP[i+1];
				IP[i].beforeFreeIP = &IP[i-1];
				IP[i].stage = IP_Free;
				IP[i].ip = dhcp.IPAdd(START_IP, i);
			}

			IP[0].nextFreeIP = &IP[1];
			IP[0].beforeFreeIP = nil;
			IP[0].stage = IP_Free;
			IP[0].ip = START_IP;

			lastFreeIP = &IP[0];
			endFreeIP = &IP[len(IP)-1];
			endReserveIP = nil;
		}//Init IP end

		var offsetToIP = func(offset int) *queueIP{
			return &IP[offset];
		};
		var getLastFreeIP = func() (freeIP *queueIP){
			{//FreeIP
				if lastFreeIP == nil { return nil; }

				if lastFreeIP.nextFreeIP != nil { lastFreeIP.nextFreeIP.beforeFreeIP = nil; }

				freeIP = lastFreeIP;
				lastFreeIP = lastFreeIP.nextFreeIP;

				if lastFreeIP == nil { endFreeIP = nil; }

				freeIP.nextFreeIP = nil;
			}
			{//ReserveIP
				freeIP.nextFreeIP = nil;
				freeIP.beforeFreeIP = endReserveIP;

				if endReserveIP != nil { endReserveIP.nextFreeIP = freeIP; }

				endReserveIP = freeIP;
				freeIP.stage = IP_Reserved;
			}

			return;
		};
		var getFreeIP = func(offset int) (freeIP *queueIP){
			{//FreeIP
				if IP[offset].beforeFreeIP != nil { IP[offset].beforeFreeIP.nextFreeIP = IP[offset].nextFreeIP; }
				if IP[offset].nextFreeIP != nil { IP[offset].nextFreeIP.beforeFreeIP = IP[offset].beforeFreeIP; }

				if &IP[offset] == endFreeIP { endFreeIP = IP[offset].beforeFreeIP; }
				if &IP[offset] == lastFreeIP { lastFreeIP = IP[offset].nextFreeIP; }

				IP[offset].nextFreeIP = nil;
				IP[offset].beforeFreeIP = nil;

			}
			freeIP = &IP[offset];
			{//ReserveIP
				freeIP.nextFreeIP = nil;
				freeIP.beforeFreeIP = endReserveIP;

				if endReserveIP != nil { endReserveIP.nextFreeIP = freeIP; }

				endReserveIP = freeIP;
				freeIP.stage = IP_Reserved;
			}

			return;
		};
		var getReserveIP = func(offset int) (freeIP *queueIP){
			{//ReserveIP
				if IP[offset].beforeFreeIP != nil { IP[offset].beforeFreeIP.nextFreeIP = IP[offset].nextFreeIP; }
				if IP[offset].nextFreeIP != nil { IP[offset].nextFreeIP.beforeFreeIP = IP[offset].beforeFreeIP; }

				if &IP[offset] == endReserveIP { endReserveIP = IP[offset].beforeFreeIP; }

				IP[offset].nextFreeIP = nil;
				IP[offset].beforeFreeIP = nil;
			}
			freeIP = &IP[offset];

			freeIP.stage = IP_NotFree;

			return;
		};
		var retNotFreeIP = func(freeIP *queueIP) {
			freeIP.nextFreeIP = nil;
			freeIP.beforeFreeIP = endFreeIP;

			if endFreeIP != nil { endFreeIP.nextFreeIP = freeIP;
			} else { lastFreeIP = freeIP; }

			endFreeIP = freeIP;
			freeIP.stage = IP_Free;
		};
		var freeIPTest = func(offset int) Stage{
			// Dead code:
			//switch{
			//	case (IP[offset].nextFreeIP == nil && IP[offset].beforeFreeIP == nil) &&
			//		 (IP[offset] != lastFreeIP /*|| IP[offset] != endFreeIP*/) &&
			//		 (IP[offset] != endReserveIP): return 2;
			//	case IP[offset].reserved: return 1;
			//	default: return 0;
			//}
			return IP[offset].stage;
		};

		var clientMAC map[net.HardwareAddr] *queueIP;

		var safeSetMAC = func(offset int, MAC net.HardwareAddr){
			if clientMAC[MAC] != nil && clientMAC[MAC] != &IP[offset] {
				if IP[offset].stage == IP_Reserved {
					getReserveIP(offset);
				}
				retNotFreeIP(&IP[offset]);
			}
			clientMAC[MAC] = &IP[offset];
			IP[offset].mac = MAC;
		};
		var setMAC = func(freeIP *queueIP, MAC net.HardwareAddr){
			clientMAC[MAC] = freeIP;
			freeIP.mac = MAC;
		};
		var deleteMAC = func(MAC net.HardwareAddr){
			clientMAC[MAC].mac = nil;
			delete(clientMAC, MAC);
		};



		return dhcp.Handler{
			ServeDHCP: func(req dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) dhcp.Packet {
				(map[dhcp.MessageType] func() dhcp.Packet{
					dhcp.Discover: func() dhcp.Packet{
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

						var reqMAC = req.CHAddr();
						var reqIP = net.IP(options[OptionRequestedIPAddress]);

						if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(net.IP{0,0,0,0}) { reqIP = tr; }

						if reqIP!=nil { if r:=dhcp.IPRange(START_IP, reqIP)-1; !(r>=0 && r<IP_Range) { reqIP=nil; } }

						switch{
						//first
						case reqIP == nil && clientMAC[reqMAC] == nil:
							var record = getLastFreeIP();	//TODO: if record == nil
							setMAC(record, reqMAC);
							return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, record.ip, time.Hour, nil);
						//double first
						case reqIP == nil && clientMAC[reqMAC] != nil && freeIPTest(clientMAC[reqMAC]) == IP_Reserved:
							return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clientMAC[reqMAC].ip, time.Hour, nil);
						//old my friend
						case reqIP == nil && clientMAC[reqMAC] != nil && freeIPTest(clientMAC[reqMAC]) == IP_NotFree:
							//change state to IP_Reserved
							if getFreeIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1) != clientMAC[reqMAC] { panic("Bug#657438310"); } //TODO: Check
							return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clientMAC[reqMAC].ip, time.Hour, nil);
						//detect error in this program
						case reqIP == nil && clientMAC[reqMAC] != nil && freeIPTest(clientMAC[reqMAC]) == IP_Free:
							panic("Bug#5464985");
							return nil;
						case reqIP != nil && clientMAC[reqMAC] == nil:
							var offset = dhcp.IPRange(START_IP, reqIP)-1;
							switch freeIPTest(offset) {
							//use user IP
							case IP_Free:
								var record = getFreeIP(offset);
								setMAC(record, reqMAC);
								return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, record.ip, time.Hour, nil);
							//try send another ip
							case IP_Reserved, IP_NotFree:
								var record = getLastFreeIP();
								setMAC(record, reqMAC);
								return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, record.ip, time.Hour, nil);
							}
						case reqIP != nil && clientMAC[reqMAC] != nil:
							var offsetReqIP = dhcp.IPRange(START_IP, reqIP)-1;
							if bytes.Equal(offsetToIP(offsetReqIP).mac, reqMAC) {
								switch freeIPTest(offsetReqIP){
								//double
								case IP_Reserved:
									return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, reqIP, time.Hour, nil);
								//my friend
								case IP_NotFree:
									//change state to IP_Reserved
									if getFreeIP(dhcp.IPRange(START_IP, reqIP)-1) != clientMAC[reqMAC] { panic("Bug#657438310+"); } //TODO: Check
									return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, reqIP, time.Hour, nil);
								//detect error in this program
								case IP_Free:
									panic("Bug#5464985+");
									return nil;
								}
							}else{
								switch freeIPTest(offsetReqIP){
								//move to user IP
								case IP_Free:
									safeSetMAC(offsetReqIP, reqMAC);
									var record = getFreeIP(offsetReqIP);
									return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, record.ip, time.Hour, nil);
								case IP_Reserved, IP_NotFree:
									switch freeIPTest(clientMAC[reqMAC]){
									//double sclerosis
									case IP_Reserved:
										return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clientMAC[reqMAC].ip, time.Hour, nil);
									//my sclerosis friend
									case IP_NotFree:
										//change state to IP_Reserved
										if getFreeIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1) != clientMAC[reqMAC] { panic("Bug#657438310-"); } //TODO: Check
										return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clientMAC[reqMAC].ip, time.Hour, nil);
									//detect error in this program
									case IP_Free:
										panic("Bug#5464985-");
										return nil;
									}
								}
							}
						}
						/*return dhcp.ReplyPacket(req, dhcp.Offer, SERVER_IP, clientMAC[reqMAC],
							h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]), nil);*/

						return nil;
					},
					dhcp.Request: func() dhcp.Packet{
						var reqMAC = req.CHAddr();
						var reqIP = net.IP(options[OptionRequestedIPAddress]);
						var reqServerIP = net.IP(options[dhcp.OptionServerIdentifier]);

						if tr:=req.CIAddr(); reqIP == nil && !tr.Equal(net.IP{0,0,0,0}) { reqIP = tr; }

						//very buggy client
						if reqServerIP == nil && reqIP == nil { return nil; }

						//try correct
						if reqServerIP == nil { reqServerIP = SERVER_IP; }
						if reqIP == nil {
							if reqIP = clientMAC[reqMAC].ip; reqIP == nil { return nil; }	//correct fail
						}

						var reqIPInRange bool; {
							r:=dhcp.IPRange(START_IP, reqIP)-1;
							reqIPInRange = r>=0 && r<IP_Range;
						}
						var serversIPEqual = reqServerIP.Equal(SERVER_IP);

						switch {
						case serversIPEqual && reqIPInRange && clientMAC[reqMAC] != nil:
							var offsetReqIP = dhcp.IPRange(START_IP, reqIP)-1;
							if bytes.Equal(offsetToIP(offsetReqIP).mac, reqMAC) {										//TODO: test speed "offsetToIP(offsetReqIP)==clientMAC[reqMAC]" && "clientMAC[reqMAC].ip.Equal(reqIP)"
								switch freeIPTest(offsetReqIP){
								case IP_Reserved:
									//change state to IP_NotFree
									if getReserveIP(offsetReqIP) != clientMAC[reqMAC] { panic("Bug#+545448411000512+"); } //TODO: Check
									return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, reqIP, time.Hour, nil);
								case IP_NotFree:
									return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, reqIP, time.Hour, nil);
								//detect error in this program
								case IP_Free:
									panic("Bug#+5464985+");
									return nil;
								}
							}else{
								switch freeIPTest(offsetReqIP){
								//move to user IP
								case IP_Free:
									safeSetMAC(offsetReqIP, reqMAC);
									var record = getFreeIP(offsetReqIP);
									record = getReserveIP(offsetReqIP);
									return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, record.ip, time.Hour, nil);
								case IP_Reserved, IP_NotFree:
									switch clientMAC[reqMAC].stage {
									case IP_Reserved:
										//delete user
										retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
										deleteMAC(reqMAC);
										return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
									case IP_NotFree:
										//delete user
										retNotFreeIP(clientMAC[reqMAC]);
										deleteMAC(reqMAC);
										return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
									//detect error in this program
									case IP_Free:
										panic("Bug#-5464985-");
										return nil;
									}
								}
							}
						case serversIPEqual && reqIPInRange && clientMAC[reqMAC] == nil:
							var offset = dhcp.IPRange(START_IP, reqIP)-1;
							switch freeIPTest(offset) {
							//use user IP
							case IP_Free:
								var record = getFreeIP(offset);
								record = getReserveIP(offset);
								setMAC(record, reqMAC);
								return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, record.ip, time.Hour, nil);
							//user broken off
							case IP_Reserved, IP_NotFree:
								return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
							}
						case serversIPEqual && !reqIPInRange && clientMAC[reqMAC] != nil:
							switch clientMAC[reqMAC].stage {
							case IP_Reserved:
								//delete user
								retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
								deleteMAC(reqMAC);
								return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
							case IP_NotFree:
								//delete user
								retNotFreeIP(clientMAC[reqMAC]);
								deleteMAC(reqMAC);
								return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
								//detect error in this program
							case IP_Free:
								panic("Bug#--5464985--");
								return nil;
							}
						case serversIPEqual && !reqIPInRange && clientMAC[reqMAC] == nil:
							return dhcp.ReplyPacket(req, dhcp.NAK, SERVER_IP, nil, 0, nil);
						case !serversIPEqual && reqIPInRange && clientMAC[reqMAC] != nil:
							var offsetReqIP = dhcp.IPRange(START_IP, reqIP)-1;
							if bytes.Equal(offsetToIP(offsetReqIP).mac, reqMAC) {										//TODO: test speed "offsetToIP(offsetReqIP)==clientMAC[reqMAC]" && "clientMAC[reqMAC].ip.Equal(reqIP)"
								switch freeIPTest(offsetReqIP){
								case IP_Reserved:
									//change state to IP_NotFree
									if getReserveIP(offsetReqIP) != clientMAC[reqMAC] { panic("Bug#++545448411000512++"); } //TODO: Check
									return nil;
								case IP_NotFree:
									return nil;
								//detect error in this program
								case IP_Free:
									panic("Bug#++5464985++");
									return nil;
								}
							}else{
								switch freeIPTest(offsetReqIP){
								//move to user IP
								case IP_Free:
									safeSetMAC(offsetReqIP, reqMAC);
									var record = getFreeIP(offsetReqIP);
									record = getReserveIP(offsetReqIP);
									return nil;
								case IP_Reserved, IP_NotFree:
									switch clientMAC[reqMAC].stage {
									case IP_Reserved:
										//delete user
										retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
										deleteMAC(reqMAC);
										return nil;
									case IP_NotFree:
										//delete user
										retNotFreeIP(clientMAC[reqMAC]);
										deleteMAC(reqMAC);
										return nil;
									//detect error in this program
									case IP_Free:
										panic("Bug#---5464985---");
										return nil;
									}
								}
							}
						case !serversIPEqual && reqIPInRange && clientMAC[reqMAC] == nil:
							var offset = dhcp.IPRange(START_IP, reqIP)-1;
							switch freeIPTest(offset) {
							//use user IP
							case IP_Free:
								var record = getFreeIP(offset);
								record = getReserveIP(offset);
								setMAC(record, reqMAC);
								return nil;
							//user broken off
							case IP_Reserved, IP_NotFree:
								return nil;
							}
						case !serversIPEqual && !reqIPInRange && clientMAC[reqMAC] != nil:
							switch clientMAC[reqMAC].stage {
							case IP_Reserved:
								//delete user
								retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
								deleteMAC(reqMAC);
								return nil;
							case IP_NotFree:
								//delete user
								retNotFreeIP(clientMAC[reqMAC]);
								deleteMAC(reqMAC);
								return nil;
							//detect error in this program
							case IP_Free:
								panic("Bug#--5464985--");
								return nil;
							}
						case !serversIPEqual && !reqIPInRange && clientMAC[reqMAC] == nil:
							return nil;
						}

						return nil;
					},
					dhcp.Decline: func() dhcp.Packet{
						var reqMAC = req.CHAddr();

						if clientMAC[reqMAC] != nil {
							log.Println("IP conflict (Decline msg) - IP: " + clientMAC[reqMAC].ip.String() + " MAC: " + reqMAC.String());

							switch clientMAC[reqMAC].stage {
							case IP_Reserved:
								//delete user
								retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
								deleteMAC(reqMAC);
								return nil;
							case IP_NotFree:
								//delete user
								retNotFreeIP(clientMAC[reqMAC]);
								deleteMAC(reqMAC);
								return nil;
							//detect error in this program
							case IP_Free:
								panic("Bug#!--5464985--!");
								return nil;
							}
						}

						return nil;
					},
					dhcp.Release: func() dhcp.Packet{
						var reqMAC = req.CHAddr();
						var reqIP  = req.CIAddr();

						if reqIP.Equal(net.IP{0,0,0,0}) { reqIP = nil; }

						if clientMAC[reqMAC] != nil && !(reqIP != nil && !clientMAC[reqMAC].ip.Equal(reqIP)) {
							switch clientMAC[reqMAC].stage {
							case IP_Reserved:
								//delete user
								retNotFreeIP(getReserveIP(dhcp.IPRange(START_IP, clientMAC[reqMAC].ip)-1));
								deleteMAC(reqMAC);
								return nil;
							case IP_NotFree:
								//delete user
								retNotFreeIP(clientMAC[reqMAC]);
								deleteMAC(reqMAC);
								return nil;
							//detect error in this program
							case IP_Free:
								panic("Bug#!!--5464985--!!");
								return nil;
							}
						}

						return nil;
					},
					dhcp.Inform: func() dhcp.Packet{
						var reqMAC = req.CHAddr();
						var reqIP  = req.CIAddr();

						if clientMAC[reqMAC].ip.Equal(reqIP) { // + "clientMAC[reqMAC] != nil" :-)
							return dhcp.ReplyPacket(req, dhcp.ACK, SERVER_IP, nil, 0, nil);
						}else{
							log.Println("Interesting (Inform msg)...");
						}

						return nil;
					},
				})[msgType]();
				return nil;
			},
		};
	}());
}