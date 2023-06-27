package server

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/gosnmp/gosnmp"
	"golang.org/x/crypto/ssh"

	"github.com/ebobo/modem_prod_go/pkg/model"
)

var filter = flag.String("filter", "icmp6 and ether", "BPF filter for capture") // Default "icmp6 and ether[6:4] & 0xffffff00 = 0x001e4200" look for teltonika devices
var iface = flag.String("iface", "eno1", "Select interface where to capture")
var snaplen = flag.Int("snaplen", 512, "Maximun size to read for each packet")
var promisc = flag.Bool("promisc", true, "Enable promiscuous mode")
var timeoutT = flag.Int("timeout", 40, "Connection Timeout in seconds")

// Constructor function to populate default values
func NewModemInfo(mac string) model.Modem {
	modemInfo := model.Modem{}
	modemInfo.MacAddress = mac
	modemInfo.IPV6 = "::"
	modemInfo.SwitchPort = -1
	modemInfo.IMEI = ""     // -i
	modemInfo.ICCID = ""    // -J
	modemInfo.IMSI = ""     // -x
	modemInfo.Firmware = "" // -y
	modemInfo.Serial = ""   // -a
	modemInfo.Model = ""    // -m
	modemInfo.Upgraded = false
	modemInfo.State = 0 // 0: unknown, 1: normal, 2: busy, 3: error
	modemInfo.FailCount = 0
	modemInfo.LastUpdated = 0
	return modemInfo
}

func RunModemService() {
	log.Println("discovery start")
	defer log.Println("discovery end")

	// Define channels for communication between goroutines
	updateModemInfoChan := make(chan model.Modem)

	// Start goroutine for discovering modems
	go modemDiscovery(updateModemInfoChan)

	time.Sleep(3 * time.Second)

	// Start goroutine for pinging ff02::1%{iface} to trick modems into letting us discover them
	go pingRoutine(*iface)

	// Start goroutine for routinely checking which port a modem is connected to
	go mapModemMAC_Port(updateModemInfoChan)

	// This should probably be a proper database later down the line
	modemList := make(map[string]model.Modem)

	var printModemList bool = false

	for {

		// Wait for updated modem info
		modemInfoReceived := <-updateModemInfoChan

		// Add/update the info in the list of discovered modems
		if modem, ok := modemList[modemInfoReceived.MacAddress]; ok {
			if modemInfoReceived.IPV6 != "::" && modemInfoReceived.IPV6 != modem.IPV6 {
				log.Printf("Updating IP address for modem with MAC %s to %s\n", modemInfoReceived.MacAddress, modemInfoReceived.IPV6)
				modem.IPV6 = modemInfoReceived.IPV6
				modem.State = 1
				modemList[modemInfoReceived.MacAddress] = modem
				printModemList = true
			}

			if modemInfoReceived.SwitchPort > -1 && modemInfoReceived.SwitchPort != modem.SwitchPort {
				modem.SwitchPort = modemInfoReceived.SwitchPort
				modemList[modemInfoReceived.MacAddress] = modem
				printModemList = true
			}

			if modemInfoReceived.State == 1 && modemInfoReceived.Upgraded { // This will need to be  changed
				fmt.Println("Modem was upgraded")
				modem.State = modemInfoReceived.State
				modem.Upgraded = modemInfoReceived.Upgraded
				modemList[modemInfoReceived.MacAddress] = modem
				printModemList = true
			}

			// Update IMEI?
			if modem.IMEI == "" && modemInfoReceived.IMEI != "" {
				log.Println("Modem's IMEI was fetched")
				modem.State = modemInfoReceived.State
				modem.IMEI = modemInfoReceived.IMEI
				modem.ICCID = modemInfoReceived.ICCID
				modem.IMSI = modemInfoReceived.IMSI
				modem.Firmware = modemInfoReceived.Firmware
				modem.Serial = modemInfoReceived.Serial
				modem.Model = modemInfoReceived.Model
				modemList[modemInfoReceived.MacAddress] = modem
				printModemList = true
			}
		} else {
			log.Printf("Adding new modem with MAC %s and IP %s\n", modemInfoReceived.MacAddress, modemInfoReceived.IPV6)
			modem := NewModemInfo(modemInfoReceived.MacAddress)
			modem.IPV6 = modemInfoReceived.IPV6
			modem.SwitchPort = modemInfoReceived.SwitchPort
			if modem.IPV6 == "::" || modem.IPV6 == "" {
				modem.State = 2 // 2: busy
			}
			modemList[modemInfoReceived.MacAddress] = modem
			printModemList = true
		}

		// Update last_updated
		modem := modemList[modemInfoReceived.MacAddress]
		modem.LastUpdated = int(time.Now().Unix())
		modemList[modemInfoReceived.MacAddress] = modem

		if printModemList {
			for k := range modemList {
				// fmt.Printf("key[%s] value[%v]\n", k, v)
				fmt.Printf("%+v\n", modemList[k])
			}
			printModemList = false
		}

		// Start different goroutines
		for i, m := range modemList {
			if m.State == 1 {
				if m.IMEI == "" {
					log.Printf("m.imei == \"%s\", therefore we need to fetch the imei\n", m.IMEI)
					m.State = 2 // 2: busy
					modemList[i] = m
					go readModemInfo(updateModemInfoChan, modemList[i])
				}
				if !m.Upgraded {
					m.State = 2
					modemList[i] = m
					go upgradeModem(updateModemInfoChan, modemList[i])
				}
			}
		}

		time.Sleep(time.Second)
	}
}

func pingRoutine(iface string) {
	var str1 string = "ff02::01%" + iface
	for {
		time.Sleep(10 * time.Second)
		// log.Println("Ping!")
		_, cmderr := exec.Command("ping", "-6", "-c 1", str1).Output()
		if cmderr != nil {
			log.Printf("pingRoutine error %s", cmderr)
		}
	}
}

func modemDiscovery(c chan<- model.Modem) {
	flag.Parse()

	var timeout time.Duration = time.Duration(*timeoutT) * time.Second

	// Opening Device
	handle, err := pcap.OpenLive(*iface, int32(*snaplen), *promisc, timeout)
	log.Println("using iface ", *iface)
	if err != nil {
		log.Fatal(err)
	}

	defer handle.Close()

	// Applying BPF Filter if it exists
	if *filter != "" {
		log.Println("applying filter ", *filter)
		err := handle.SetBPFFilter(*filter)
		if err != nil {
			log.Fatalf("error applying BPF Filter %s - %v", *filter, err)
		}
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for packet := range packetSource.Packets() {

		if ethernetLayer := packet.Layer(layers.LayerTypeEthernet); ethernetLayer != nil {

			eth, _ := ethernetLayer.(*layers.Ethernet)

			if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {

				ipv6, _ := ipv6Layer.(*layers.IPv6)

				macStr := eth.SrcMAC[:].String()
				ip6Str := ipv6.SrcIP[:].String()

				sendModemInfo := NewModemInfo(macStr)
				sendModemInfo.IPV6 = ip6Str
				c <- sendModemInfo

			}
		}
	}
}

func upgradeModem(c chan<- model.Modem, m model.Modem) {

	log.Printf("Upgrading %s", m.MacAddress)
	time.Sleep(time.Duration(rand.Intn(4)+3) * time.Second)
	m.State = 1
	m.Upgraded = true
	log.Printf("Finished upgrading %s", m.MacAddress)
	c <- m
}

func readModemInfo(c chan<- model.Modem, modem model.Modem) {
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("admin"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	modemIP_String := "[" + modem.IPV6 + "%" + *iface + "]:22"

	client, err := ssh.Dial("tcp", modemIP_String, config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}

	log.Printf("Dialed modem %s", modemIP_String)

	modem.IMEI = readModemInfoRunCommand(client, "gsmctl -i")
	modem.ICCID = readModemInfoRunCommand(client, "gsmctl -J")
	modem.IMSI = readModemInfoRunCommand(client, "gsmctl -x")
	modem.Firmware = readModemInfoRunCommand(client, "gsmctl -y")
	modem.Serial = readModemInfoRunCommand(client, "gsmctl -a")
	modem.Model = readModemInfoRunCommand(client, "gsmctl -m")
	modem.State = 1

	time.Sleep(time.Duration(rand.Intn(4)+3) * time.Second)

	c <- modem
}

func readModemInfoRunCommand(client *ssh.Client, command string) string {

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	res := "Command error"
	if err := session.Run(command); err != nil {
		log.Fatal("Failed to run: " + err.Error())
	}
	if strings.HasPrefix(b.String(), "Failed") || strings.HasPrefix(b.String(), "ERROR") {
		res = "N/A"
	} else {
		res = strings.ReplaceAll(b.String(), "\r", "")
		res = strings.ReplaceAll(res, "\n", "")
	}
	// log.Printf("Command %s returned %s, function returning %s", command, b.String(), res)
	return res
}

func mapModemMAC_Port(c chan<- model.Modem) {
	// Struct for temp storage og SNMPwalk results
	for {
		type WalkInfo struct {
			oid  string
			mac  string
			port string
		}

		walkList := make(map[string]WalkInfo)

		// SNMP device parameters
		ip := "192.168.2.1"
		community := "public"

		// Create an SNMP Go client
		snmpClient := &gosnmp.GoSNMP{
			Target:    ip,
			Port:      161,
			Community: community,
			Version:   gosnmp.Version2c, // Specify the SNMP version here
			Timeout:   time.Duration(2) * time.Second,
		}

		// Establish an SNMP connection
		err := snmpClient.Connect()
		if err != nil {
			log.Fatalf("SNMP Connect failed: %v", err)
		}
		defer snmpClient.Conn.Close()

		// First we get the MAC addresses of connected devices

		// Build the SNMP walk OID for MAC info
		oid_mac := ".1.2.6.1.2.1.18.4.3.3.3"

		// Perform the SNMP walk
		results, err := snmpClient.WalkAll(oid_mac)
		if err != nil {
			log.Fatalf("SNMP Walk failed: %v", err)
		}

		// Process the SNMP walk results
		for _, pdu := range results {
			macStringRaw := fmt.Sprintf("%v", pdu.Value)                                            // Convert weird type (interface{}) to string
			macStringTrim := strings.Split(strings.Trim(strings.Trim(macStringRaw, "]"), "["), " ") // Remove brackets and split into a slice of strings
			macByte := []byte{0, 0, 0, 0, 0, 0}                                                     // declare and initialize
			for n, stringByte := range macStringTrim {
				macByteInt, err := strconv.Atoi(stringByte)
				if err != nil {
					fmt.Println("Error during conversion")
					return
				}
				macByte[n] = byte(macByteInt)
			}
			// Format the MAC address byte array into a string
			macAddress := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", macByte[0], macByte[1], macByte[2], macByte[3], macByte[4], macByte[5])

			macWalk := WalkInfo{}
			macWalk.mac = macAddress
			macWalk.oid = strings.TrimPrefix(pdu.Name, oid_mac)
			walkList[strings.TrimPrefix(pdu.Name, oid_mac)] = macWalk
		}

		// Build the SNMP walk OID for port info
		oid_port := ".1.5.8.1.5.9.10.4.3.1.2"

		// Perform the SNMP walk
		results, err = snmpClient.WalkAll(oid_port)
		if err != nil {
			log.Fatalf("SNMP Walk failed: %v", err)
		}

		// Process the SNMP walk results
		for _, pdu := range results {
			portWalk := walkList[strings.TrimPrefix(pdu.Name, oid_port)]
			portWalk.port = fmt.Sprintf("%v", pdu.Value)
			walkList[strings.TrimPrefix(pdu.Name, oid_port)] = portWalk

		}

		// for _, walk := range walkList {
		// 	fmt.Printf("MAC %s is connected to port %v\n", walk.mac, walk.port)
		// }

		for _, walk := range walkList {
			if strings.HasPrefix(walk.mac, "00:1f:43:") {
				// fmt.Printf("Teltonika device with MAC %s is connected to port %v\n", walk.mac, walk.port)
				sendModemInfo := NewModemInfo(walk.mac)
				sendModemInfo.SwitchPort, err = strconv.Atoi(walk.port)
				if err != nil {
					fmt.Println("Error getting switch port")
					return
				}
				c <- sendModemInfo
			}
		}
		time.Sleep(10 * time.Second)
	}

}
