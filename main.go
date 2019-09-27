package main

import (
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/ctcpip/notifize"
	vcard "github.com/emersion/go-vcard"
	"github.com/mediocregopher/radix/v3"
)

const (
	redisURL             = "redis://127.0.0.1:6379/5"
	redisPhoneNumbersKey = "PDPhoneNumbers"
)

var (
	pool *radix.Pool
)

func init() {
	var err error
	pool, err = radix.NewPool("tcp", redisURL, 10)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	freshNumbers, err := downloadVcfCardAndGetPhoneNumbers()
	if err != nil {
		log.Fatal(err)
	}

	storedNumbersString, err := getNumbersFromRedis()
	if err != nil {
		log.Fatal(err)
	}

	freshNumbersString := strings.Join(freshNumbers, "|")
	if storedNumbersString != freshNumbersString {
		err = storePhoneNumbers(freshNumbersString)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func downloadVcfCardAndGetPhoneNumbers() (numbers []string, err error) {
	url := "https://s3.amazonaws.com/pdpartner/PagerDuty+Outgoing+Numbers.vcf"
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	decoder := vcard.NewDecoder(resp.Body)

	numbers = []string{}

	for {
		card, serr := decoder.Decode()
		if serr == io.EOF {
			break
		} else if serr != nil {
			err = serr
			return
		}

		for _, number := range card.Values(vcard.FieldTelephone) {
			numbers = append(numbers, number)
		}
	}

	sort.Strings(numbers)

	return
}

func getNumbersFromRedis() (numbers string, err error) {
	err = pool.Do(radix.Cmd(&numbers, "GET", redisPhoneNumbersKey))
	if err != nil {
		return
	}

	return
}

func storePhoneNumbers(numbers string) (err error) {
	notifize.Display("Pager Duty VCF Checker", "Phone Numbers changed. Please download new ones", true, "")

	err = pool.Do(radix.Cmd(nil, "SET", redisPhoneNumbersKey, numbers))
	return
}
