package main

import(
	"fmt"
	"log"
	"net"
	"strconv"

	dhcp "github.com/krolaw/dhcp4"
)

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


//TODO: config to JSON
//TODO: exclude return (or set H._START_IP and range_ip) .0/24 (IP & mask = all zero) & .255/24 (IP & mask = all one) IPs
//TODO: check delta tLeaseEnd > 2*(40sec + 60sec + 20sec)
//TODO: создать отдельный пакет CLI/main и qDHCPv4
//TODO: test speed: Go-style (struct) vs JavaScript-style (func+closure)

//TODO: выдавать нормальные опции (шлюз, маска, ...)

//TODO: Q: network IO is async?  A: No.


func main() {
	fmt.Printf("Hello LAN!");

	var conn ServeConn;
	var deviceChanHandlers map[[2]byte] chan<-reqChan;
	var defaultChanHandler              chan<-reqChan;

	// rule: len(device)==n && len(chan)==n
	{
		var dhcpOptions = dhcp.Options{
			dhcp.OptionSubnetMask:       []byte{255, 255,  0, 0},
			dhcp.OptionRouter:           []byte{194, 188, 64, 8},
			dhcp.OptionDomainNameServer: []byte{194, 188, 64, 8},
		}

		var tLeaseEnd = []int16{
			11*60 + 35,
			12*60 + 50,
			13*60 + 05,
		};

		// for parallel run rewrite to new(DhcpHandler) OR (better) store in DhcpHandler{} const fields and
		// create (new()) in Init() writable fields
		var Handlers = make([]struct{cReq chan <-reqChan; hDhcp DhcpHandler}, 4);
		{ cTemp := make(chan reqChan, 4); Handlers[0].cReq = cTemp; Handlers[0].hDhcp.Init(&conn, net.IP{194, 188, 64, 28}, net.IP{194, 188, 32, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }
		{ cTemp := make(chan reqChan, 4); Handlers[1].cReq = cTemp; Handlers[1].hDhcp.Init(&conn, net.IP{194, 188, 64, 28}, net.IP{194, 188, 36, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }
		{ cTemp := make(chan reqChan, 4); Handlers[2].cReq = cTemp; Handlers[2].hDhcp.Init(&conn, net.IP{194, 188, 64, 28}, net.IP{194, 188, 40, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }
		{ cTemp := make(chan reqChan, 4); Handlers[3].cReq = cTemp; Handlers[3].hDhcp.Init(&conn, net.IP{194, 188, 64, 28}, net.IP{194, 188, 44, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }

		deviceChanHandlers = map[[2]byte] chan <-reqChan{
			[2]byte{'a', 'n'}: Handlers[1].cReq,
			[2]byte{'i', 'P'}: Handlers[2].cReq,
			[2]byte{'W', 'i'}: Handlers[3].cReq,
		};
		defaultChanHandler = Handlers[0].cReq;


		go Handlers[0].hDhcp.GoHandle();
		go Handlers[1].hDhcp.GoHandle();
		go Handlers[2].hDhcp.GoHandle();
		go Handlers[3].hDhcp.GoHandle();
	}

	for{
		log.Println(func() error{
			var l, err = net.ListenPacket("udp4", ":67");
			if err != nil {
				return err;
			}
			defer l.Close();

			conn = l;
			var buffer = make([]byte, 1500);
			for{
				var n, addr, err = conn.ReadFrom(buffer);
				if err != nil { return err; }
				if n < 240    { continue; }      // Packet too small to be DHCP

				var req = dhcp.Packet(buffer[:n]);
				if req.HLen() > 16 { continue; } // Invalid size

				var reqType dhcp.MessageType;
				var options = req.ParseOptions();
				if t := options[dhcp.OptionDHCPMessageType]; len(t) != 1 { continue;
				}else{
					reqType = dhcp.MessageType(t[0]);
					if reqType < dhcp.Discover || reqType > dhcp.Inform  { continue; }
				}

				/*log.Println(leases[string(bytes.SplitN(options[dhcp.OptionHostName],[]byte{'-'},2)[0])]);*/
				var device [2]byte; copy(device[:],options[dhcp.OptionHostName][:2]);

				if     deviceChanHandler,exist := deviceChanHandlers[device]; exist==true {
					   deviceChanHandler  <- reqChan{req, reqType, options, addr};
				}else{ defaultChanHandler <- reqChan{req, reqType, options, addr}; }
			}
		}())}
}

func serveOut(conn ServeConn, req dhcp.Packet, res dhcp.Packet, addr net.Addr) error {
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
		if _, e := conn.WriteTo(res, addr); e != nil {
			return e;
		}
	}
	return nil;
}