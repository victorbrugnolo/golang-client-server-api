package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/mattn/go-sqlite3"
)

type API struct {
	DolarPrice string `yaml:"dolar-price"`
}

type Config struct {
	API API `yaml:"api"`
}

type USDBRL struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type DolarPrice struct {
	USDBRL USDBRL `json:"USDBRL"`
}

type ResponseDTO struct {
	Price string `json:"price"`
}

func main() {
	http.HandleFunc("/cotacao", GetDolarPriceHandler)
	http.ListenAndServe(":8080", nil)
}

func GetDolarPriceHandler(w http.ResponseWriter, r *http.Request) {
	config := Config{}
	config.Load()

	db, err := gorm.Open(sqlite.Open("dolar-price.db"), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&USDBRL{})

	dolarPrice, err := GetDolarPrice(config.API.DolarPrice, r.Context())

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = SaveDolarPriceOnDatabase(dolarPrice, db, r.Context())

	if err != nil {
		return
	}

	err = SaveDolarPriceOnFile(dolarPrice)

	if err != nil {
		return
	}

	json.NewEncoder(w).Encode(ResponseDTO{Price: dolarPrice.USDBRL.Bid})
}

func GetDolarPrice(url string, ctx context.Context) (*DolarPrice, error) {
	log.Println("Getting dolar price...")

	getContext, cancel := context.WithTimeout(ctx, 200*time.Millisecond)

	defer cancel()

	req, err := http.NewRequestWithContext(getContext, "GET", url, nil)
	// time.Sleep(300 * time.Second)

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println("Getting dolar price failed, error: ", err.Error())

		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var dolarPrice DolarPrice

	err = json.Unmarshal(body, &dolarPrice)

	if err != nil {
		return nil, err
	}

	log.Println("Dolar price retrieved")
	return &dolarPrice, nil

}

func (c *Config) Load() *Config {
	configFile, err := os.ReadFile("config/application-local.yaml")

	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(configFile, c)

	if err != nil {
		panic(err)
	}

	return c
}

func SaveDolarPriceOnDatabase(dolarPrice *DolarPrice, db *gorm.DB, ctx context.Context) error {
	saveContext, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err := db.WithContext(saveContext).Create(&dolarPrice.USDBRL).Error

	if err != nil {
		log.Println("Saving dolar price on db failed, error: ", err.Error())
	}

	log.Println("Dolar price saved on database")

	return err
}

func SaveDolarPriceOnFile(dolarPrice *DolarPrice) error {
	f, err := os.Create("cotacao.txt")

	if err != nil {
		panic(err)
	}

	_, err = f.Write([]byte("Dólar: " + dolarPrice.USDBRL.Bid + "\n"))

	if err != nil {
		log.Println("Error writing to file: ", err.Error())
	}

	f.Close()

	log.Println("Dolar price saved on file")

	return err
}
