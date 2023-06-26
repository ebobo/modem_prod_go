package utility

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ebobo/modem_prod_go/pkg/model"
)

func MakeDirIfNotExists(dirpath string) error {
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		err := os.Mkdir(dirpath, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func GenerateFakeModems(num int) []model.Modem {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	fakeModems := make([]model.Modem, num)
	for i := 0; i < num; i++ {
		// state := rand.Intn(4) // 0: unknown, 1: normal, 2: busy, 3: error
		progress := 0
		// if state == 2 {
		// 	progress = int(rand.Intn(100))
		// }
		fakeModems[i] = model.Modem{
			MacAddress:  generateMacAddress(),
			IPV6:        generateIPV6(),
			SwitchPort:  i + 1,
			Model:       "TRB-140",
			State:       1,
			Firmware:    "TRB1_R_00.07.04.2",
			Serial:      uuid.New().String(),
			Kernel:      "5.4.221",
			Upgraded:    false,
			LastUpdated: int(time.Now().Unix()),
			FailCount:   0,
			SIMProvider: "Twilio",
			SIMStatus:   false,
			IMEI:        generateIMEI(),
			ICCID:       generateICCID(),
			IMSI:        generateIMSI(),
			Progress:    progress,
		}
	}
	return fakeModems
}

func generateIPV6() string {
	b := make([]string, 8)
	for i := 0; i < 8; i++ {
		b[i] = fmt.Sprintf("%04x", rand.Intn(0x10000)) // 0x10000 is hexadecimal for 65536, which provides 4 digits of randomness.
	}
	return strings.Join(b, ":")
}

func generateIMEI() string {
	return fmt.Sprintf("%013d", rand.Intn(1e13))
}

func generateIMSI() string {
	raw := fmt.Sprintf("%015d", rand.Intn(1e15))
	split := make([]string, 0, 4)
	for i := 0; i < len(raw); i += 3 {
		end := i + 3
		if end > len(raw) {
			end = len(raw)
		}
		split = append(split, raw[i:end])
	}
	return strings.Join(split, " ")
}

func generateICCID() string {
	section1 := rand.Int63n(1e10) // Generate a 10-digit number
	section2 := rand.Int63n(1e10) // Generate another 10-digit number
	return fmt.Sprintf("%010d%010d", section1, section2)
}

func generateMacAddress() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		rand.Intn(255), rand.Intn(255), rand.Intn(255),
		rand.Intn(255), rand.Intn(255), rand.Intn(255))
}
