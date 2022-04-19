package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"io"
	"log"
	"net/http"
)

type Currency struct {
	Text     string `xml:",chardata" json:"Text"`
	ID       string `xml:"ID,attr" json:"ID"`
	NumCode  string `xml:"NumCode" json:"NumCode"`
	CharCode string `xml:"CharCode" json:"CharCode"`
	Nominal  string `xml:"Nominal" json:"Nominal"`
	Name     string `xml:"Name" json:"Name"`
	Value    string `xml:"Value" json:"Value"`
}
type ExchangeRates struct {
	XMLName    xml.Name   `xml:"ValCurs" json:"-"`
	Text       string     `xml:",chardata" json:"Text"`
	Date       string     `xml:"Date,attr" json:"Date"`
	Name       string     `xml:"name,attr" json:"Name"`
	Currencies []Currency `xml:"Valute" json:"ValCurs"`
}

func (rates *ExchangeRates) Print() {
	log.Printf("Data: %s", rates.Date)
	for _, c := range rates.Currencies {
		log.Printf("%s: %s", c.CharCode, c.Value)
	}
}

func (rates *ExchangeRates) toJson() ([]byte, error) {
	return json.Marshal(rates)
}

func (rates *ExchangeRates) toXML() ([]byte, error) {
	return xml.Marshal(rates)
}

func getXML(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error: %v", resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func GetExchangeRates(buf []byte) (ExchangeRates, error) {
	ExRates := ExchangeRates{}
	d := xml.NewDecoder(bytes.NewReader(buf))
	d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unknown charset: %s", charset)
		}
	}

	err := d.Decode(&ExRates)
	if err != nil {
		return ExchangeRates{}, err
	}

	return ExRates, nil
}

func IndexPage(w http.ResponseWriter, r *http.Request) {
	s := "CBRF to json/xml in UTF-8\n"
	w.Write([]byte(s))
}

func CBRFResponser(url string) ExchangeRates {
	xmlBytes, err := getXML(url)
	if err != nil {
		log.Printf("Failed to get XML: %v", err)
	}
	data, err := GetExchangeRates(xmlBytes)
	if err != nil {
		log.Println(err)
	}
	return data
}

func getUrl(r *http.Request) string {
	if l := r.FormValue("lang"); l == "eng" {
		return fmt.Sprintf("https://www.cbr.ru/scripts/XML_daily_%s.asp", l)
	} else {
		return "https://www.cbr.ru/scripts/XML_daily.asp"
	}
}

func getDate(r *http.Request) string {
	if d := r.FormValue("date_req"); len(d) > 0 {
		return fmt.Sprintf("date_req=%s", d)
	}
	return ""
}

func MakeUrl(r *http.Request) string {
	return fmt.Sprintf("%s?%s", getUrl(r), getDate(r))
}

func CBRFtoJson(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	url := MakeUrl(r)

	log.Println("URL", url)

	data := CBRFResponser(url)

	jsonResp, err := data.toJson()
	if err != nil {
		log.Println(err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func CBRFtoXML(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusCreated)

	url := MakeUrl(r)

	log.Println("URL", url)

	data := CBRFResponser(url)
	jsonResp, err := data.toXML()
	if err != nil {
		log.Println(err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", IndexPage)
	mux.HandleFunc("/index", IndexPage)
	mux.HandleFunc("/cbrf/json", CBRFtoJson)
	mux.HandleFunc("/cbrf/xml", CBRFtoXML)

	serv := http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: mux,
	}
	log.Println("Start listening...")
	serv.ListenAndServe()
}