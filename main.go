package main

import(
	"fmt"
	"log"
	"net"
	"time"
	"strings"
	"strconv"
	"encoding/binary"
	"errors"
	"encoding/json"
	"io/ioutil"

	"qDHCPv4/utils/closure"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/ZiroKyl/reflectDHCP"
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


//TODO: создать отдельный пакет CLI/main и qDHCPv4
//TODO: test speed: Go-style (struct) vs JavaScript-style (func+closure)

//TODO: Q: network IO is async?  A: No.


type TimeM int16;

func (t *TimeM) UnmarshalJSON(b []byte) error{
	tp, err := time.Parse("15:04", strings.Trim(string(b), `"`)); //magic string format: stackoverflow.com/a/14106561

	if err == nil {
		*t = TimeM(tp.Hour()*60 + tp.Minute());
	}

	return err;
}

type IPv4byte []byte;

func (ipb *IPv4byte) UnmarshalJSON(b []byte) error{
	str := strings.Trim(string(b), `"`)

	if ip := net.ParseIP(str); ip != nil {
		if ip=ip.To4(); ip != nil {
			*ipb = []byte(ip);
			return nil;
		}
	}

	return errors.New("IPv4 is not correct: " + str);
}


//TODO: config to JSON -> http://stackoverflow.com/a/21610752 http://golang.org/pkg/encoding/json/ -> struct for dhcpOptions
func configure(jsonConf []byte, conn *ServeConn) (err error, deviceChanHandlers map[[2]byte] chan<-reqChan, defaultChanHandler chan<-reqChan){
	type Device struct{
		Name    string   `json:"name"`;
		StartIP IPv4byte `json:"startIP"`;
		RangeIP int      `json:"rangeIP"`;
	};
	var conf = struct{
		GlobalOptions  reflectDHCP.Options `json:"globalOptions"`;
		LeaseEndTime []TimeM               `json:"leaseEndTime"`;
		Devices      []Device              `json:"devices"`;
		DefaultDevice  Device              `json:"defaultDevice"`;
	}{GlobalOptions: reflectDHCP.Options{}};	//reflectDHCP.Options is map

	if err = json.Unmarshal(jsonConf, &conf); err != nil {
		return;
	}

	{// check input conf
		// check delta LeaseEndTime > 2*(40sec + 120sec + 20sec)
		if lt := len(conf.LeaseEndTime); lt >= 1 {
			for i := 1; i < lt; i++ {
				if !(conf.LeaseEndTime[i-1]+6 /*Minute*/ < conf.LeaseEndTime[i]) {
					err = errors.New("(LeaseEndTime #" + string(i) + " - LeaseEndTime #" + string(i-1) + ") not > 5 minute");
					return;
				}
			}
		}else { err = errors.New("leaseEndTime must have at least one element"); return; }

		// check Default Device
		if len(conf.DefaultDevice.Name) != 0 { log.Println("defaultDevice.name is ignored! pls. delete the defaultDevice.name"); }
		if conf.DefaultDevice.RangeIP    < 1 { err = errors.New("defaultDevice.rangeIP must be >= 1"); return; }

		// check Devices
		for i, dev := range conf.Devices {
			if len(dev.Name) != 2 { err = errors.New("length of devices[" + string(i) + "].name must be = 2"); return; }
			if dev.RangeIP    < 1 { err = errors.New("defaultDevice.rangeIP must be >= 1"); return; }
		}

		{// IP ranges intersection check
			var ipr = make([]interface{}, 1+len(conf.Devices));
			ipr[0] = []int{int(binary.BigEndian.Uint32(conf.DefaultDevice.StartIP)), conf.DefaultDevice.RangeIP -1/*not need "-1" in future*/};

			for i,d := range conf.Devices {
				ipr[i+1] = []int{int(binary.BigEndian.Uint32(d.StartIP)), d.RangeIP -1/*not need "-1" in future*/};
			}

			closure.Sort(ipr, func(i0, i1 int)bool{ return ipr[i0].([]int)[0] < ipr[i1].([]int)[0]; });

			for i:=0; i<len(ipr)-1; i++ {
				if a,add,b := ipr[i  ].([]int)[0],
				              ipr[i  ].([]int)[1],
				              ipr[i+1].([]int)[0];
				!(a+add < b) {
					var bufIPa = make(net.IP,4); binary.BigEndian.PutUint32(bufIPa,uint32(a));
					var bufIPb = make(net.IP,4); binary.BigEndian.PutUint32(bufIPb,uint32(b));

					err = errors.New("IP ranges intersection! startIP: " + bufIPa.String() + " AND " + bufIPb.String());
					return;
				}
			}

			//TODO: exclude return (or set H._START_IP and range_ip) .0/24 (IP & mask = all zero) & .255/24 (IP & mask = all one) IPs
		}
	}

	// >:-()
	var tLeaseEnd = make([]int16, len(conf.LeaseEndTime));
	for i := range conf.LeaseEndTime{
		tLeaseEnd[i] = int16(conf.LeaseEndTime[i]);
	}

	// for parallel run rewrite to new(DhcpHandler) OR (better) store in DhcpHandler{} const fields and
	// create (new()) in Init() writable fields
	var Handlers = make([]struct{cReq chan <-reqChan; hDhcp DhcpHandler}, 1+len(conf.Devices));

	// rule: len(device)==n && len(chan)==n
	for i := range conf.Devices {
		cTemp := make(chan reqChan, 1+len(conf.Devices));
		Handlers[1+i].cReq = cTemp;
		Handlers[1+i].hDhcp.Init(conn, net.IP(conf.GlobalOptions[dhcp.OptionServerIdentifier]),
			                            net.IP(conf.Devices[i].StartIP),
			                            conf.Devices[i].RangeIP,
			                            dhcp.Options(conf.GlobalOptions),
			                            tLeaseEnd,
			                            cTemp);
	}
	{
		cTemp := make(chan reqChan, 1+len(conf.Devices));
		Handlers[0].cReq = cTemp;
		Handlers[0].hDhcp.Init(conn, net.IP(conf.GlobalOptions[dhcp.OptionServerIdentifier]),
			                          net.IP(conf.DefaultDevice.StartIP),
			                          conf.DefaultDevice.RangeIP,
			                          dhcp.Options(conf.GlobalOptions),
			                          tLeaseEnd,
			                          cTemp);
	}

	deviceChanHandlers = make(map[[2]byte] chan <-reqChan, len(conf.Devices));

	for i := range conf.Devices {
		var devName [2]byte; copy(devName[:],conf.Devices[i].Name[:2]);

		deviceChanHandlers[devName] = Handlers[1+i].cReq;
	}
	defaultChanHandler = Handlers[0].cReq;

	for i := range Handlers {
		go Handlers[i].hDhcp.GoHandle();
	}

	return;
}

func main() {
	fmt.Printf("Hello LAN!");

	var conn ServeConn;
	var deviceChanHandlers map[[2]byte] chan<-reqChan;
	var defaultChanHandler              chan<-reqChan;

	var configJSON, err = ioutil.ReadFile("src/qDHCPv4/config.json");
	if err != nil {
		log.Fatalln("error reading config file:", err);
	}

	err,deviceChanHandlers,defaultChanHandler = configure(configJSON, &conn);
	if err != nil {
		log.Fatalln("error in config file:", err);
	}

	// rule: len(device)==n && len(chan)==n
	/*{
		var dhcpOptions = dhcp.Options{
			dhcp.OptionSubnetMask:       []byte{255, 255,  0, 0},
			dhcp.OptionRouter:           []byte{194, 188, 64, 8},
			dhcp.OptionDomainNameServer: []byte{194, 188, 64, 8},
		}

		var tLeaseEnd = []int16{
			11*60 + 35,
			12*60 + 50,
			17*60 + 23,
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
	}*/

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
				var device [2]byte;
				if len(options[dhcp.OptionHostName])>=2 { copy(device[:],options[dhcp.OptionHostName][:2]); }

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
