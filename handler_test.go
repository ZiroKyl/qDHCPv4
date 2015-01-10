package main

import(
	"testing"
	"net"

	dhcp "github.com/krolaw/dhcp4"
)

var SERVER_IP = net.IP{194, 188, 64, 28};

func output(hDhcp *DhcpHandler, msgType dhcp.MessageType, mac net.HardwareAddr, IAddr net.IP, options dhcp.Options) dhcp.Packet{
	return hDhcp.Handler(dhcp.RequestPacket(msgType, mac, IAddr, []byte{1,2,3,4}, true, options.SelectOrderOrAll(nil)), msgType, options);
}

type tsOut struct{
	msgType dhcp.MessageType;
	startIP net.IP;
	rangeIP int;
};

func outCheck(out dhcp.Packet, cOut tsOut, testN int, t *testing.T){
	if out == nil {
		if cOut.rangeIP == 0 { t.Log("#", testN, "| nil"); return;
		}else{ t.Fatal("out is nil"); }
	}

	var options  = out.ParseOptions();
	var msgType  = dhcp.MessageType(options[dhcp.OptionDHCPMessageType][0]);
	var serverIP = net.IP          (options[dhcp.OptionServerIdentifier]);
	var getIP    = out.YIAddr();
	var ipOffset = 0;
	if cOut.startIP != nil { ipOffset = dhcp.IPRange(cOut.startIP, getIP)-1; }

	if msgType != cOut.msgType                                        { t.Fatal("#", testN, "|", "msgType is",     msgType ); }
	if !serverIP.Equal(SERVER_IP)                                     { t.Fatal("#", testN, "|", "outServerIP is", serverIP); }
	if cOut.rangeIP!=0 && !(ipOffset >= 0 && ipOffset < cOut.rangeIP) { t.Fatal("#", testN, "|", "ipOffset is",    ipOffset); }

	t.Log("#", testN, "|", msgType, serverIP, getIP);
}

func TestDhcpHandler_0(t *testing.T){
	type tsIn struct{
		msgType    dhcp.MessageType;
		mac        net.HardwareAddr;
		reqIP    []byte;
		IAddr      net.IP;
		serverIP []byte;
	};
	type testSamples struct{
		in  tsIn;
		out tsOut;
	};

	var smpls = [...]testSamples{
		{//======dhcp.Discover======// reqIP vs IAddr
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,1}, reqIP: nil, IAddr: []byte{192,168,1,52} },
			out:tsOut{ msgType: dhcp.Offer,                                                       startIP: net.IP{194,188,32,1}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,2}, reqIP: []byte{192,168,1,52}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,2}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,3}, reqIP: nil, IAddr: []byte{194,188,32,4} },
			out:tsOut{ msgType: dhcp.Offer,                                                       startIP: net.IP{194,188,32,4}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,4}, reqIP: []byte{194,188,32,5}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,5}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,5}, reqIP: []byte{194,188,32,6}, IAddr: []byte{194,188,32,7} },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,6}, rangeIP: 1 },
		},
		{//first
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,6}, reqIP: nil, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer, startIP: net.IP{194,188,32,3}, rangeIP: 1 },
		},
		{//double first
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,1}, reqIP: nil, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                                       startIP: net.IP{194,188,32,1}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,2}, reqIP: nil, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,2}, rangeIP: 1 },
		},{//old my friend
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,2}, reqIP: nil, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,2}, rangeIP: 1 },
		},
		{//use user IP
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,7}, reqIP: []byte{194,188,32,7}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,7}, rangeIP: 1 },
		},{//try send another lease
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,7}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,8}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,7}, reqIP: []byte{194,188,32,7}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,7}, rangeIP: 1 },
		},{//try send another lease
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,9}, reqIP: []byte{194,188,32,7}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,9}, rangeIP: 1 },
		},
		{//double
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,9}, reqIP: []byte{194,188,32,9}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,9}, rangeIP: 1 },
		},{//my friend
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,7}, reqIP: []byte{194,188,32,7}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,7}, rangeIP: 1 },
		},{//move to user IP
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,9}, reqIP: []byte{194,188,32,10}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,10}, rangeIP: 1 },
		},{//double sclerosis
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,10}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,8},  rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,8}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,8}, rangeIP: 1 },
		},{//my sclerosis friend
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,5}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,8},  rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,9}, reqIP: []byte{194,188,32,10}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,10}, rangeIP: 1 },
		},{//double sclerosis
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,10}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,8},  rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,4}, reqIP: []byte{194,188,32,5}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,5}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,8}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,32,8}, rangeIP: 1 },
		},{//my sclerosis friend
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,50,8}, reqIP: []byte{194,188,32,5}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,32,8},  rangeIP: 1 },
		},
		{//======dhcp.Request======// reqIP vs IAddr
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,1},             reqIP: []byte{194,188,33,1}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                                       startIP: net.IP{194,188,33,1}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,1}, reqIP: nil, IAddr: []byte{194,188,33,1}, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                                         startIP: net.IP{194,188,33,1}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,2}, reqIP: []byte{194,188,33,2}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,2}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,2}, reqIP: []byte{194,188,33,2}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,2}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,3}, reqIP: []byte{194,188,33,3}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,3}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,3}, reqIP: []byte{194,188,33,3}, IAddr: []byte{194,188,33,4}, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,3}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,3}, reqIP: []byte{194,188,33,4}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,4}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,3}, reqIP: []byte{194,188,33,3}, IAddr: []byte{194,188,33,4}, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,3}, rangeIP: 1 },
		},
		{//very buggy client
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,4}, reqIP: nil, IAddr: nil },
			out:tsOut{ rangeIP: 0 },
		},
		{//try correct
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,5}, reqIP: []byte{194,188,33,5}, IAddr: nil },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,5}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,6}, reqIP: []byte{194,188,33,6}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,6}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,6}, reqIP: nil, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,6}, rangeIP: 1 },
		},{//correct fail
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,7}, reqIP: nil, IAddr: nil },
			out:tsOut{ rangeIP: 0 },
		},
		{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,8}, reqIP: []byte{194,188,33,8}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,8}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,8}, reqIP: []byte{194,188,33,8}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,8}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,8}, reqIP: []byte{194,188,33,8}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,8}, rangeIP: 1 },
		},
		{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,9}, reqIP: []byte{194,188,33,9}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,9}, rangeIP: 1 },
		},{//move to user IP
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,9}, reqIP: []byte{194,188,33,10}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,33,10}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,10},reqIP: []byte{194,188,33,9}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,9}, rangeIP: 1 },
		},{//delete user
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,9}, reqIP: []byte{194,188,33,9}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,9}, reqIP: []byte{194,188,33,10}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,33,10}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,10}, reqIP: []byte{194,188,33,9}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                              startIP: net.IP{194,188,33,9}, rangeIP: 1 },
		},{//delete user
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,9}, reqIP: []byte{194,188,33,9}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},
		{//use user IP
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,11}, reqIP: []byte{194,188,33,11}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                              startIP: net.IP{194,188,33,11}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,12}, reqIP: []byte{194,188,33,12}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                            startIP: net.IP{194,188,33,12}, rangeIP: 1 },
		},{//user broken off
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,13}, reqIP: []byte{194,188,33,11}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{//user broken off
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,13}, reqIP: []byte{194,188,33,12}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{//delete user
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,11}, reqIP: []byte{19 ,188,33,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{//delete user
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,12}, reqIP: []byte{19 ,188,33,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},
		{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,14}, reqIP: []byte{19 ,188,33,1}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},
		{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,60,15}, reqIP: []byte{19 ,188,33,1}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                            startIP: net.IP{194,188,32,1}, rangeIP: 1024 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,60,15}, reqIP: []byte{19 ,188,33,1}, IAddr: nil, serverIP: []byte{19,188,33,200} },
			out:tsOut{ rangeIP: 0 },
		},
		{//======dhcp.Decline======// leases conflict
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,70,0}, reqIP: []byte{194,188,34,0}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,34,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,70,0}, reqIP: []byte{194,188,34,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,34,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Decline,  mac: net.HardwareAddr{10,20,30,40,70,0}, reqIP: nil, IAddr: nil },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,70,1}, reqIP: []byte{194,188,34,0}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,34,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,70,1}, reqIP: []byte{194,188,34,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,34,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,70,0}, reqIP: []byte{194,188,34,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},
		{//======dhcp.Release======//
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,80,0}, reqIP: []byte{194,188,35,0}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,35,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,80,0}, reqIP: []byte{194,188,35,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,35,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Release,  mac: net.HardwareAddr{10,20,30,40,80,0}, reqIP: nil, IAddr: []byte{194,188,35,0} },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,80,1}, reqIP: []byte{194,188,35,0}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,35,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,80,1}, reqIP: []byte{194,188,35,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,35,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Release,  mac: net.HardwareAddr{10,20,30,40,80,2}, reqIP: nil, IAddr: []byte{194,188,35,0} },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,80,2}, reqIP: []byte{194,188,35,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{
			in: tsIn { msgType: dhcp.Release,  mac: net.HardwareAddr{10,20,30,40,80,1}, reqIP: nil, IAddr: []byte{194,188,35,1} },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,80,2}, reqIP: []byte{194,188,35,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.NAK },
		},{
			in: tsIn { msgType: dhcp.Release,  mac: net.HardwareAddr{10,20,30,40,80,1}, reqIP: nil, IAddr:nil },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Request,  mac: net.HardwareAddr{10,20,30,40,80,2}, reqIP: []byte{194,188,35,0}, IAddr: nil, serverIP: SERVER_IP },
			out:tsOut{ msgType: dhcp.ACK,                                             startIP: net.IP{194,188,35,0}, rangeIP: 1 },
		},
		{//======dhcp.Inform======//
			in: tsIn { msgType: dhcp.Discover, mac: net.HardwareAddr{10,20,30,40,90,0}, reqIP: []byte{194,188,36,0}, IAddr: nil },
			out:tsOut{ msgType: dhcp.Offer,                                           startIP: net.IP{194,188,36,0}, rangeIP: 1 },
		},{
			in: tsIn { msgType: dhcp.Inform,   mac: net.HardwareAddr{10,20,30,40,90,0}, reqIP: nil, IAddr: []byte{194,188,36,0} },
			out:tsOut{ msgType: dhcp.ACK },
		},{
			in: tsIn { msgType: dhcp.Inform,   mac: net.HardwareAddr{10,20,30,40,90,0}, reqIP: nil, IAddr: []byte{194,188,36,1} },
			out:tsOut{ rangeIP: 0 },
		},{
			in: tsIn { msgType: dhcp.Inform,   mac: net.HardwareAddr{10,20,30,40,90,1}, reqIP: nil, IAddr: []byte{194,188,36,0} },
			out:tsOut{ msgType: dhcp.ACK },
		},
	}


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

	var hDhcp DhcpHandler;
	{ cTemp := make(chan reqChan, 4); hDhcp.Init(nil, net.IP{194, 188, 64, 28}, net.IP{194, 188, 32, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }

	for i,s := range smpls {
		outCheck(output(&hDhcp, s.in.msgType, s.in.mac, s.in.IAddr, dhcp.Options{
			dhcp.OptionRequestedIPAddress: s.in.reqIP,
			dhcp.OptionServerIdentifier:   s.in.serverIP,
		}), s.out, i, t);
	}

	//show error t.Log(smpls[n]);
}

func BenchmarkDhcpHandler_0(b *testing.B){	//-benchtime 0.005s: 5000 ns/op 1230 B/op 11 allocs/op
											//                   6010 ns/op 1230 B/op 13 allocs/op
	var mac  = net.HardwareAddr{1, 2, 3, 4, 5, 6};
	var opt  = dhcp.Options{};
	var opts = opt.SelectOrderOrAll(nil);
	var xId  = []byte{1,2,3,4};

	var packet = dhcp.RequestPacket(dhcp.Discover, mac, nil, xId, true, opts);

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

	var hDhcp DhcpHandler;
	{ cTemp := make(chan reqChan, 4); hDhcp.Init(nil, net.IP{194, 188, 64, 28}, net.IP{194, 188, 32, 1}, 1024, dhcpOptions, tLeaseEnd, cTemp); }

	b.ResetTimer();
	for i:=0;i<b.N;i++ {
		//output(dhcp.Discover, mac, nil, opt);
		hDhcp.Handler(packet, dhcp.Discover, opt);
		mac[0]++;
		packet.SetCHAddr(mac);
	}
}


//TODO: map vs switch-case