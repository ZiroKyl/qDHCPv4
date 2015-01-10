// Modified version of github.com/krolaw/dhcp4/server.go

package main

import(
	"net"
	"strconv"

	dhcp "github.com/krolaw/dhcp4"
)

func serveIn(conn *ServeConn, deviceChanHandlers map[[2]byte] chan<-reqChan, defaultChanHandler chan<-reqChan) error{
	var l, err = net.ListenPacket("udp4", ":67");
	if err != nil {
		return err;
	}
	defer l.Close();

	*conn = l;

	var buffer = make([]byte, 1500);
	for{
		var n, addr, err = l.ReadFrom(buffer);
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
		var device [2]byte;
		if len(options[dhcp.OptionHostName])>=2 { copy(device[:],options[dhcp.OptionHostName][:2]); }

		if     deviceChanHandler,exist := deviceChanHandlers[device]; exist==true {
			   deviceChanHandler  <- reqChan{req, reqType, options, addr};
		}else{ defaultChanHandler <- reqChan{req, reqType, options, addr}; }
	}
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